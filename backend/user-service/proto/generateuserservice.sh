protoc \
--go_out=./user-service-gen --go_opt=paths=source_relative \
--go-grpc_out=./user-service-gen --go-grpc_opt=paths=source_relative \
--grpc-gateway_out=./user-service-gen --grpc-gateway_opt=paths=source_relative ./user-service.proto