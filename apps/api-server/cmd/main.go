package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"speech.local/apps/api-server/internal/di"
	"speech.local/packages/config"
	"speech.local/packages/telemetry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	telemetryCfg := telemetry.Config{}
	if err := config.LoadConfig(&telemetryCfg); err != nil {
		log.Fatalf("failed to load telemetry config: %v", err)
	}

	shutdown, err := telemetry.Init(telemetryCfg)
	if err != nil {
		log.Fatalf("failed to initialize telemetry: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("failed to shutdown tracer: %v", err)
		}
	}()

	logger, err := telemetry.NewLogger("api-server")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	taskHandler, err := di.InitializeTaskDependencies()
	if err != nil {
		log.Fatalf("failed to initialize dependencies: %v", err)
	}

	r := gin.Default()
	r.Use(otelgin.Middleware(telemetryCfg.ServiceName))
	r.Use(telemetry.LoggerMiddleware(logger))

	meter := otel.GetMeterProvider().Meter("api-server")
	r.Use(telemetry.MetricsMiddleware(meter))

	api := r.Group("/api")
	{
		tasks := api.Group("/tasks")
		{
			tasks.POST("/confirm", taskHandler.HandleConfirmTask)
			tasks.GET("/:id", taskHandler.HandleGetTask)
			tasks.GET("/:id/stream", taskHandler.HandleStreamSummary)
		}
		api.GET("/upload-url", taskHandler.HandleGetUploadURL)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
