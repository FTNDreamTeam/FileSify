package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	pb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/server"
	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {

	// Silo Storage Setup
	endpoint := os.Getenv("SILO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ROOT_USER")
	secretKey := os.Getenv("MINIO_ROOT_PASSWORD")
	userServicePort := os.Getenv("USER_SERVICE_PORT")
	mongoUri := os.Getenv("METADATA_URI")
	metaDBName := os.Getenv("META_DB")

	var store *storage.SiloStore
	var err error

	// Register Services
	//User Service
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userConn, err := grpc.NewClient("user-service:"+userServicePort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	userClient := userpb.NewUserServiceClient(userConn)
	//Metadata DB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))

	if err != nil {
		log.Fatalf("Failed to initialize mongo:%v", err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping Mongo:%v", err)
	}

	fileMetaColl := mongoClient.Database(metaDBName).Collection("file_meta")
	folderMetaColl := mongoClient.Database(metaDBName).Collection("folder_meta")

	// Keep trying to connect to Silo until successful
	for {
		store, err = storage.NewSiloStore(endpoint, accessKey, secretKey, false)
		if err == nil {
			ctx := context.Background()
			err = store.CreateBucket(ctx, "filesify")
			if err == nil {
				break
			}
		}

		log.Printf("WARNING: Failed to connect to Silo: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}
	log.Println("Successfully connected to Silo!")

	// gRPC Server Setup
	port := os.Getenv("STORAGE_PORT")

	// Keep trying to bind to the port until successful
	var lis net.Listener
	for {
		lis, err = net.Listen("tcp", ":"+port)
		if err == nil {
			break
		}
		log.Printf("WARNING: Failed to listen on port %s: %v. Retrying in 5 seconds...", port, err)
		time.Sleep(5 * time.Second)
	}

	grpcServer := grpc.NewServer()

	storageServer := server.NewStorageServer(store, userClient, fileMetaColl, folderMetaColl)

	pb.RegisterStorageServiceServer(grpcServer, storageServer)
	reflection.Register(grpcServer)

	log.Printf("gRPC Data Storage Service server is running on port :%s", lis.Addr().String())

	// Start the gRPC server
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("gRPC server failed unexpectedly: %v", err)
	}
}
