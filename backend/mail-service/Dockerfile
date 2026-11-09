FROM golang:1.27.3-alpine AS builder

WORKDIR /app

COPY ./gen ./gen

COPY ./mail-service/go.mod ./mail-service/go.sum ./mail-service/

WORKDIR /app/mail-service
RUN go mod download

COPY ./mail-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o /mail-service .

FROM alpine:latest

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /mail-service /mail-service

EXPOSE 8083 1025

ENTRYPOINT ["/mail-service"]