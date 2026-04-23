package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"speech.local/apps/api-server/internal/usecase"
	"speech.local/packages/telemetry"
)

type TaskHandler struct {
	usecase usecase.TaskUseCase
	logger  *zap.Logger
	debug   bool
}

func NewTaskHandler(usecase usecase.TaskUseCase, logger *zap.Logger, debug bool) *TaskHandler {
	return &TaskHandler{usecase: usecase, logger: logger, debug: debug}
}

type ConfirmTaskRequest struct {
	S3Key string `json:"s3_key" binding:"required"`
}

type ConfirmTaskResponse struct {
	TaskID uint   `json:"task_id"`
	Status string `json:"status"`
}

type GetUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	S3Key     string `json:"s3_key"`
}

func (h *TaskHandler) HandleGetUploadURL(c *gin.Context) {
	ext := c.Query("ext")
	if ext == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'ext' is required"})
		return
	}

	contentType := c.Query("content_type")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'content_type' is required"})
		return
	}

	log := telemetry.WithTraceID(c.Request.Context(), h.logger)
	log.Info("HandleGetUploadURL: received request",
		zap.String("ext", ext),
		zap.String("content_type", contentType),
	)

	uploadURL, s3Key, err := h.usecase.GetAudioUploadURL(c.Request.Context(), ext, contentType)
	if err != nil {
		if err == usecase.ErrInvalidFileExtension {
			log.Error("HandleGetUploadURL: invalid file extension",
				zap.String("ext", ext),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file extension"})
			return
		}
		log.Error("HandleGetUploadURL: storage error",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info("HandleGetUploadURL: success",
		zap.String("s3_key", s3Key),
	)

	c.JSON(http.StatusOK, GetUploadURLResponse{
		UploadURL: uploadURL,
		S3Key:     s3Key,
	})
}

func (h *TaskHandler) HandleMockUpload(c *gin.Context) {
	if !h.debug {
		c.JSON(http.StatusNotFound, gin.H{"error": "mock upload endpoint not found"})
		return
	}

	log := telemetry.WithTraceID(c.Request.Context(), h.logger)
	log.Info("HandleMockUpload: simulating upload success",
		zap.String("s3key", c.Param("s3key")),
	)

	taskID, err := h.usecase.ConfirmTask(c.Request.Context(), c.Param("s3key"))
	if err != nil {
		log.Error("HandleMockUpload: failed to confirm task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info("HandleMockUpload: task created", zap.Uint("task_id", taskID))

	c.JSON(http.StatusOK, gin.H{
		"status":  "uploaded",
		"task_id": taskID,
	})
}

func (h *TaskHandler) HandleConfirmTask(c *gin.Context) {
	var req ConfirmTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: s3_key is required"})
		return
	}

	log := telemetry.WithTraceID(c.Request.Context(), h.logger)
	log.Info("HandleConfirmTask: received request",
		zap.String("s3_key", req.S3Key),
	)

	taskID, err := h.usecase.ConfirmTask(c.Request.Context(), req.S3Key)
	if err != nil {
		log.Error("HandleConfirmTask: error",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info("HandleConfirmTask: success",
		zap.Uint("task_id", taskID),
	)

	c.JSON(http.StatusAccepted, ConfirmTaskResponse{
		TaskID: taskID,
		Status: "PENDING",
	})
}

func (h *TaskHandler) HandleGetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	task, err := h.usecase.GetTaskDetail(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) HandleStreamSummary(c *gin.Context) {
	taskIDStr := c.Param("task_id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_id"})
		return
	}

	ch, closeFunc, err := h.usecase.StreamTaskSummary(c.Request.Context(), uint(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer closeFunc()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-ch; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}
