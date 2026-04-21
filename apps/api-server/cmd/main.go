package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"speech.local/apps/api-server/internal/di"
)

func main() {
	taskHandler, err := di.InitializeTaskDependencies()
	if err != nil {
		log.Fatalf("failed to initialize dependencies: %v", err)
	}

	r := gin.Default()
	r.GET("/tasks/upload-url", taskHandler.HandleGetUploadURL)
	r.POST("/tasks/confirm", taskHandler.HandleConfirmTask)

	r.Run(":8080")
}
