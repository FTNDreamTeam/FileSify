package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/server"
	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
	pb "github.com/FTNDreamTeam/FileSify/backend/gen/data-storage-service-gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Silo Storage Setup
	endpoint := os.Getenv("SILO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ROOT_USER")
	secretKey := os.Getenv("MINIO_ROOT_PASSWORD")

	// Initialize Silo Object Store
	store, err := storage.NewSiloStore(endpoint, accessKey, secretKey, false)
	if err != nil {
		log.Fatalf("Failed to initialize Silo client: %v", err)
	}
	// Ensure the bucket exists
	ctx := context.Background()
	if err := store.CreateBucket(ctx, "filesify"); err != nil {
		log.Fatalf("Failed to ensure bucket exists: %v", err)
	}
	log.Println("Successfully connected to Silo!")

	// gRPC Server Setup
	port := os.Getenv("STORAGE_PORT")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("Failed to listen on port %s: %v", port, err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()

	// Register Services
	storageServer := server.NewStorageServer(store)
	pb.RegisterStorageServiceServer(grpcServer, storageServer)
	reflection.Register(grpcServer)

	log.Printf("gRPC Data Storage Service server is running on port :%s", lis.Addr().String())

	// Start the gRPC server
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("gRPC server failed unexpectedly: %v", err)
		os.Exit(1)
	}
}
