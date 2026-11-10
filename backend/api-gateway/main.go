package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	pb "gen/gateway"
	userpb "gen/user-service-gen"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedGatewayServiceServer
	userClient userpb.UserServiceClient
}

func (s *server) RegisterUser(ctx context.Context, request *userpb.RegisterUserRequest) (*userpb.RegisterUserResponse, error) {
	return s.userClient.RegisterUser(ctx, request)
}

func (s *server) LoginUser(ctx context.Context, request *userpb.LoginUserRequest) (*userpb.LoginUserResponse, error) {
	return s.userClient.LoginUser(ctx, request)
}

func (s *server) VerifyAccount(ctx context.Context, request *userpb.VerifyAccountRequest) (*userpb.VerifyAccountResponse, error) {
	return s.userClient.VerifyAccount(ctx, request)
}

func main() {
	// Internal gRPC listener (only reachable from inside the Docker network)
	lis, err := net.Listen("tcp", "api-gateway:"+os.Getenv("GATEWAY_PORT"))
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Dial user-service
	userConn, err := grpc.NewClient(
		"user-service:"+os.Getenv("USER_SERVICE_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Failed to dial user-service:", err)
	}
	userClient := userpb.NewUserServiceClient(userConn)

	// Start internal gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterGatewayServiceServer(grpcServer, &server{userClient: userClient})
	reflection.Register(grpcServer)
	go func() {
		log.Fatalln(grpcServer.Serve(lis))
	}()

	// Dial the gateway's own gRPC server, so gwmux can forward REST → gRPC
	gwConn, err := grpc.NewClient(
		"api-gateway:"+os.Getenv("GATEWAY_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Failed to dial gateway gRPC:", err)
	}

	gwmux := runtime.NewServeMux()
	if err := pb.RegisterGatewayServiceHandler(context.Background(), gwmux, gwConn); err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}

	// ONE public port: /api/* goes to gwmux, everything else to the file server
	root := http.NewServeMux()
	root.Handle("/api/", gwmux)
	root.Handle("/", http.FileServer(http.Dir("./frontend")))

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: root,
	}

	log.Println("Serving frontend + API on http://0.0.0.0:8080")
	log.Fatalln(httpServer.ListenAndServe())
}
