FROM golang:1.27.3-alpine AS builder

WORKDIR /app

COPY ./backend/gen ./backend/gen

COPY ./frontend ./frontend

COPY ./backend/api-gateway/go.mod ./backend/api-gateway/go.sum ./backend/api-gateway/

WORKDIR /app/backend/api-gateway
RUN go mod download

COPY ./backend/api-gateway/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/backend/api-gateway/api-gateway /app/api-gateway

COPY --from=builder /app/frontend /app/frontend

EXPOSE 8081 8090 8080

ENTRYPOINT [ "./api-gateway" ]