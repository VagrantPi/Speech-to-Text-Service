FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY apps/llm-worker/go.mod apps/llm-worker/go.sum ./
COPY packages ./packages
COPY go.work ./go.work
COPY go.work.sum ./go.work.sum

# Download dependencies
RUN go mod download

# Copy source code
COPY apps/llm-worker ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Run the binary
CMD ["./main"]
