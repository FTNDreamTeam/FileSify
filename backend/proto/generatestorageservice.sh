protoc \
--go_out=../gen/data-storage-service-gen --go_opt=paths=source_relative \
--go-grpc_out=../gen/data-storage-service-gen --go-grpc_opt=paths=source_relative \
--grpc-gateway_out=../gen/data-storage-service-gen --grpc-gateway_opt=paths=source_relative ./data-storage-service.proto