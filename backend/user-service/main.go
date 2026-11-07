package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	userpb "github.com/FTNDreamTeam/FileSify/backend/user-service/proto/user-service-gen"
	"github.com/FTNDreamTeam/FileSify/backend/user-service/server"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoUri := os.Getenv("MONGO_URI")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB client: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Successfully connected to MongoDB!")

	userCollection := client.Database("users_metadata").Collection("users")

	lis, err := net.Listen("tcp", ":"+os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("Failed to listen on port 8082: %v", err)
	}

	grpcServer := grpc.NewServer()
	userServer := server.NewUserServer(userCollection)

	userpb.RegisterUserServiceServer(grpcServer, userServer)
	reflection.Register(grpcServer)

	log.Println("gRPC User Service server is running on port :" + lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
