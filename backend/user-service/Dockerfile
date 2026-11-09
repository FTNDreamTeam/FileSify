FROM golang:1.27.3-alpine AS builder

WORKDIR /app

COPY ./gen ./gen

COPY ./user-service/go.mod ./user-service/go.sum ./user-service/

WORKDIR /app/user-service
RUN go mod download

COPY ./user-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service .

FROM alpine:latest

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /user-service /user-service

EXPOSE 8082

ENTRYPOINT ["/user-service"]