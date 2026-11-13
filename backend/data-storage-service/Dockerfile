FROM golang:1.27.3-alpine AS builder

WORKDIR /app

COPY ./gen /gen

COPY ./data-storage-service/go.mod ./data-storage-service/go.sum ./

RUN go mod download

COPY ./data-storage-service/ .

RUN go build -o data-storage-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/data-storage-service .
CMD ["./data-storage-service"]