package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	url := os.Getenv("MQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	queue := os.Args[1]
	if queue == "" {
		log.Fatal("Usage: go run test_publish.go <stt-queue|llm-queue>")
	}

	payload := map[string]interface{}{
		"task_id": 1,
		"s3_key":  "uploads/audio/test-file.mp3",
	}

	if queue == "llm-queue" {
		payload = map[string]interface{}{
			"task_id":    1,
			"transcript": "這是測試用的逐字稿內容。",
		}
	}

	payloadBytes, _ := json.Marshal(payload)

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	err = ch.PublishWithContext(context.Background(),
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payloadBytes,
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}

	log.Printf("✅ Published to %s: %s", queue, payloadBytes)
}
