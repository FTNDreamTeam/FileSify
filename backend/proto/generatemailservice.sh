protoc \
--go_out=../gen/mail-service --go_opt=paths=source_relative \
--go-grpc_out=../gen/mail-service --go-grpc_opt=paths=source_relative \
--grpc-gateway_out=../gen/mail-service --grpc-gateway_opt=paths=source_relative ./mail-service.proto