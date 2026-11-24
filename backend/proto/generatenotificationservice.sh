protoc \
--go_out=../gen/notification-service-gen --go_opt=paths=source_relative \
--go-grpc_out=../gen/notification-service-gen --go-grpc_opt=paths=source_relative \
--grpc-gateway_out=../gen/notification-service-gen --grpc-gateway_opt=paths=source_relative ./notification-service.proto