FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Copy workspace files first
COPY go.work go.work.sum ./
COPY apps ./apps
COPY packages ./packages

# Download dependencies (uses go.work modules)
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./apps/api-server/cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/apps/api-server/.env ./.env

EXPOSE 8080

CMD ["./main"]
