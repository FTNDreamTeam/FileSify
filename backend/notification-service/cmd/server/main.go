package main

import (
	"log"
	"net"
	"time"

	"github.com/gocql/gocql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	notificationpb "gen/notification-service-gen"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/config"
	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/repo"
	"github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/service"
	grpcTransport "github.com/FTNDreamTeam/FileSify/backend/notification-service/internal/transport/grpc"
)

func main() {
	configs := config.LoadConfigs()

	// Setup Cassandra
	cluster := gocql.NewCluster(configs.CASSANDRA_HOST)
	cluster.Keyspace = configs.KEYSPACE
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 5 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to connect to Cassandra: %v", err)
	}
	defer session.Close()

	// Dial user-service for token validation
	userConn, err := grpc.NewClient(
		"user-service:"+configs.USER_SERVICE_PORT,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to dial user-service: %v", err)
	}
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	// Setup Layers (DI)
	notificationRepo := repo.NewNotificationRepo(session)
	notificationService := service.NewNotificationService(notificationRepo, userClient)
	notificationHandler := grpcTransport.NewNotificationHandler(notificationService)

	// Start gRPC Server
	lis, err := net.Listen("tcp", ":"+configs.PORT)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	notificationpb.RegisterNotificationServiceServer(grpcServer, notificationHandler)
	reflection.Register(grpcServer)

	log.Printf("Notification Service listening on port %s", configs.PORT)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
