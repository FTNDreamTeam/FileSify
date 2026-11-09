package main

import (
	"log"
	"mailService/server"
	"net"
	"os"

	mailpb "gen/mail-service"
	userpb "gen/user-service-gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	lis, err := net.Listen("tcp", ":"+os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("Failed to listen on port 8082: %v", err)
	}

	userConn, err := grpc.NewClient(
		"user-service:"+os.Getenv("USER_SERVICE_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	userClient := userpb.NewUserServiceClient(userConn)

	grpcServer := grpc.NewServer()
	mailServer := server.NewMailServer(userClient)

	mailpb.RegisterMailServiceServer(grpcServer, mailServer)
	reflection.Register(grpcServer)

	log.Println("gRPC Mail Service server is running on port :" + lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
