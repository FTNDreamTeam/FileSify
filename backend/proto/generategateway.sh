protoc \
--go_out=../gen/gateway --go_opt=paths=source_relative \
--go-grpc_out=../gen/gateway --go-grpc_opt=paths=source_relative \
--grpc-gateway_out=../gen/gateway --grpc-gateway_opt=paths=source_relative ./gateway.proto