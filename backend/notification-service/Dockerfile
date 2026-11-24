FROM golang:1.27.3-alpine AS builder

WORKDIR /app

# Copy the gen module (from the root context)
COPY ./backend/gen ./backend/gen

# Copy the notification service go.mod and go.sum
COPY ./backend/notification-service/go.mod ./backend/notification-service/go.sum ./backend/notification-service/

WORKDIR /app/backend/notification-service
RUN go mod download

# Copy the rest of the notification service
COPY ./backend/notification-service/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service-bin ./cmd/server/main.go

FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /notification-service-bin /notification-service-bin

EXPOSE 8084

CMD ["/notification-service-bin"]