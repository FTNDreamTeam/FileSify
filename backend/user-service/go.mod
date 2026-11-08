module github.com/FTNDreamTeam/FileSify/backend/user-service

go 1.27.3

require (
	gen v0.0.1
	go.mongodb.org/mongo-driver v1.17.10
	golang.org/x/crypto v0.58.0
	google.golang.org/genproto v0.0.0-20261104224025-d1e00e9f33ab
	google.golang.org/grpc v1.85.0
)

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261019180522-ce908d29299b // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace gen => ../gen
