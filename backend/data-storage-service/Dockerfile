FROM golang:1.27.3-alpine AS builder

WORKDIR /app

COPY ./gen ./gen

COPY ./data-storage-service/go.mod ./data-storage-service/go.sum ./data-storage-service/

WORKDIR /app/data-storage-service
RUN go mod download

COPY ./data-storage-service/ .

RUN CGO_ENABLED=0 GOOG=linux go build -o /data-storage-service .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /data-storage-service /data-storage-service

EXPOSE 8083 9000

CMD ["/data-storage-service"]