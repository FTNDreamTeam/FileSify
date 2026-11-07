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

func main() {

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("./frontend")))
	go func() { http.ListenAndServe(":8080", mux) }()

	lis, err := net.Listen("tcp", "api-gateway:"+os.Getenv("GATEWAY_PORT"))
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	userConn, err := grpc.NewClient(
		"user-service:"+os.Getenv("USER_SERVICE_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	userClient := userpb.NewUserServiceClient(userConn)

	s := grpc.NewServer()

	pb.RegisterGatewayServiceServer(s, &server{userClient: userClient})

	log.Println("Serving gRPC on 0.0.0.0:8081")
	reflection.Register(s)
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}

	gwmux := runtime.NewServeMux()
	gwConn, err := grpc.NewClient(
		"api-gateway:"+os.Getenv("GATEWAY_PORT"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	err = pb.RegisterGatewayServiceHandler(context.Background(), gwmux, gwConn)
	if err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}

	gwRest := &http.Server{
		Addr:    ":8090",
		Handler: gwmux,
	}

	log.Println("Serving gRPC-Gateway on http://0.0.0.0:8090")
	log.Fatalln(gwRest.ListenAndServe())

}
