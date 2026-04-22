FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY apps ./apps
COPY packages ./packages

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./apps/stt-worker/cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .

CMD ["./main"]
