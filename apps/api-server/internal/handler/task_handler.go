package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"speech.local/apps/api-server/internal/usecase"
)

type TaskHandler struct {
	usecase usecase.TaskUseCase
}

func NewTaskHandler(usecase usecase.TaskUseCase) *TaskHandler {
	return &TaskHandler{usecase: usecase}
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

	// TODO: 1. Redis 做單一 ip 與用戶限流

	// TODO 2. 如果音訊檔太大需要切割

	uploadURL, s3Key, err := h.usecase.GetAudioUploadURL(c.Request.Context(), ext, contentType)
	if err != nil {
		if err == usecase.ErrInvalidFileExtension {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file extension"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetUploadURLResponse{
		UploadURL: uploadURL,
		S3Key:     s3Key,
	})
}

func (h *TaskHandler) HandleConfirmTask(c *gin.Context) {
	var req ConfirmTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: s3_key is required"})
		return
	}

	taskID, err := h.usecase.ConfirmTask(c.Request.Context(), req.S3Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, ConfirmTaskResponse{
		TaskID: taskID,
		Status: "PENDING",
	})
}
