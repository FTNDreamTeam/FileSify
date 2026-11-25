FROM golang:1.27.3-alpine AS builder

WORKDIR /app

# Copy the gen module
COPY ./gen ./gen

# Tidy and add missing dependency to gen module
WORKDIR /app/gen
RUN go get google.golang.org/genproto/googleapis/type/date && go mod tidy -go=1.27.3

# Copy the notification service go.mod and go.sum
WORKDIR /app
COPY ./notification-service/go.mod ./notification-service/go.sum ./notification-service/

WORKDIR /app/notification-service
RUN go mod download

# Copy the rest of the notification service
COPY ./notification-service/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service-bin ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /notification-service-bin /notification-service-bin
EXPOSE 8084
CMD ["/notification-service-bin"]