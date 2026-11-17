package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	storagepb "gen/data-storage-service-gen"
	mailpb "gen/mail-service"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/user-service/logger"
	"github.com/FTNDreamTeam/FileSify/backend/user-service/server"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger.Init("/app/logs/user-service.log")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoUri := os.Getenv("MONGO_URI")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		log.Printf("Failed to initialize MongoDB client: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Printf("Failed to ping MongoDB: %v", err)
	}

	userCollection := client.Database("users_metadata").Collection("users")
	verificationCollection := client.Database("authentication_db").Collection("verification_mail")
	magicLinkCollection := client.Database("authentication_db").Collection("magic_link_tokens")

	if err := server.EnsureTTLIndex(ctx, verificationCollection); err != nil {
		log.Printf("Verification TTL index creation error: %v", err)
	}
	if err := server.EnsureTTLIndex(ctx, magicLinkCollection); err != nil {
		log.Printf("Magic link TTL index creation error: %v", err)
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Printf("JWT_SECRET is not set")
	}

	lis, err := net.Listen("tcp", ":"+os.Getenv("PORT"))
	if err != nil {
		log.Printf("Failed to listen on port 8082: %v", err)
	}

	mailConn, err := grpc.NewClient(
		"mail-service:"+os.Getenv("MAIL_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Failed to dial user-service:", err)
	}
	mailClient := mailpb.NewMailServiceClient(mailConn)

	twoFactorService := server.NewTFAService(redis.NewClient(&redis.Options{Addr: "redis-otp:" + os.Getenv("REDIS_PORT"), Username: "", Password: ""}), []byte(os.Getenv("HMAC_SERCRET")))

	storageConn, err := grpc.NewClient(
		"filesify-storage:"+os.Getenv("STORAGE_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	storageClient := storagepb.NewStorageServiceClient(storageConn)

	grpcServer := grpc.NewServer()
	userServer := server.NewUserServer(userCollection, verificationCollection, magicLinkCollection, mailClient, twoFactorService, storageClient)
	userpb.RegisterUserServiceServer(grpcServer, userServer)
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("Failed to serve gRPC: %v", err)
	}
}
