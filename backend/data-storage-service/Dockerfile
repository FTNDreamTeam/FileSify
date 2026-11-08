FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o data-storage-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/data-storage-service .

EXPOSE 8083

CMD ["./data-storage-service"]