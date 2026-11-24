FROM golang:1.27.3-alpine AS builder

WORKDIR /app

# Copy the gen module (from the root context)
COPY ./gen ./gen

# Copy the notification service go.mod and go.sum
COPY ./notification-service/go.mod ./notification-service/go.sum ./notification-service/

WORKDIR /app/notification-service
RUN go mod download

# Copy the rest of the notification service
COPY ./notification-service/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service-bin ./cmd/server/main.go

FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /notification-service-bin /notification-service-bin

EXPOSE 8084

CMD ["/notification-service-bin"]