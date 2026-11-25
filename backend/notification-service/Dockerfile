FROM golang:1.27.3-alpine AS builder

WORKDIR /app

# 1. Copy the gen module (Now fully self-contained)
COPY ./gen ./gen

# 2. Copy the notification service go.mod and go.sum
COPY ./notification-service/go.mod ./notification-service/go.sum ./notification-service/

# 3. Set working directory and download dependencies
WORKDIR /app/notification-service
RUN go mod download

# 4. Copy the rest of the code
COPY ./notification-service/ .

# 5. Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service-bin ./cmd/server/main.go

# --- Final lightweight stage ---
FROM alpine:latest
WORKDIR /app
COPY --from=builder /notification-service-bin /notification-service-bin
EXPOSE 8084
CMD ["/notification-service-bin"]