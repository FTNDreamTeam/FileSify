package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
)

func main() {
	// Endpoint: "localhost:9000" when running on host,
	// "silo:9000" when running inside the compose network.
	endpoint := os.Getenv("SILO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ROOT_USER")
	secretKey := os.Getenv("MINIO_ROOT_PASSWORD")
	if accessKey == "" || secretKey == "" {
		log.Fatal("MINIO_ROOT_USER and MINIO_ROOT_PASSWORD must be set")
	}

	store, err := storage.NewSiloStore(endpoint, accessKey, secretKey, false)
	if err != nil {
		log.Fatalf("Failed to create Silo store: %v", err)
	}
	log.Println("Connected to Silo at", endpoint)

	ctx := context.Background()

	bucketName := "filesify"
	if err := store.CreateBucket(ctx, bucketName); err != nil {
		log.Fatalf("Failed to ensure bucket exists: %v", err)
	}
	log.Printf("Bucket %s is ready", bucketName)

	fmt.Println("Data Storage Service is running. Press Ctrl+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down Data Storage Service gracefully...")
}
