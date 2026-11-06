package main

import (
	"context"
	"log"
	"net"
	"net/http"

	pb "github.com/FTNDreamTeam/FileSify/backend/api-gateway/proto/gateway"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedGatewayServiceServer
}

func (s *server) GetUser(ctx context.Context, request *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return &pb.GetUserResponse{Users: "Fuck you"}, nil
}

func main() {

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("../../frontend")))
	go func() { http.ListenAndServe(":8080", mux) }()

	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	s := grpc.NewServer()
	pb.RegisterGatewayServiceServer(s, &server{})

	log.Println("Serving gRPC on 0.0.0.0:8081")
	reflection.Register(s)
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	conn, err := grpc.NewClient(
		":8081",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}

	gwmux := runtime.NewServeMux()
	//gwmux.HandlePath("GET", "/api/users", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) { })
	err = pb.RegisterGatewayServiceHandler(context.Background(), gwmux, conn)
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
