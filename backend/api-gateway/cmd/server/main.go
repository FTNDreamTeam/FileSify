package main

import (
	"context"
	"log"
	"net"
	"net/http"

	httpTransport "github.com/FTNDreamTeam/FileSify/backend/api-gateway/internal/transport/http"

	storagepb "gen/data-storage-service-gen"
	pb "gen/gateway"
	notificationpb "gen/notification-service-gen"
	userpb "gen/user-service-gen"

	config "github.com/FTNDreamTeam/FileSify/backend/api-gateway/internal/config"
	service "github.com/FTNDreamTeam/FileSify/backend/api-gateway/internal/service"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedGatewayServiceServer
	userClient         userpb.UserServiceClient
	storageClient      storagepb.StorageServiceClient
	notificationClient notificationpb.NotificationServiceClient
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

func (s *server) RequestMagicLink(ctx context.Context, request *userpb.RequestMagicLinkRequest) (*userpb.RequestMagicLinkResponse, error) {
	return s.userClient.RequestMagicLink(ctx, request)
}

func (s *server) VerifyMagicLink(ctx context.Context, request *userpb.VerifyMagicLinkRequest) (*userpb.LoginUserResponse, error) {
	return s.userClient.VerifyMagicLink(ctx, request)
}

func (s *server) ChangePassword(ctx context.Context, request *userpb.ChangePasswordRequest) (*userpb.ChangePasswordResponse, error) {
	return s.userClient.ChangePassword(ctx, request)
}
func (s *server) Verify2FA(ctx context.Context, request *userpb.Verify2FARequest) (*userpb.Verify2FAResponse, error) {
	return s.userClient.Verify2FA(ctx, request)
}

func (s *server) CreateFolder(ctx context.Context, request *storagepb.CreateFolderRequest) (*storagepb.CreateFolderResponse, error) {
	return s.storageClient.CreateFolder(ctx, request)
}

func (s *server) ListFiles(ctx context.Context, request *storagepb.ListFilesRequest) (*storagepb.ListFilesResponse, error) {
	return s.storageClient.ListFiles(ctx, request)
}

func (s *server) RenameFolder(ctx context.Context, request *storagepb.RenameFolderRequest) (*storagepb.RenameFolderResponse, error) {
	return s.storageClient.RenameFolder(ctx, request)
}
func (s *server) RenameFile(ctx context.Context, request *storagepb.RenameFileRequest) (*storagepb.RenameFileResponse, error) {
	return s.storageClient.RenameFile(ctx, request)
}

func (s *server) ListNotifications(ctx context.Context, request *notificationpb.ListNotificationsRequest) (*notificationpb.ListNotificationsResponse, error) {
	return s.notificationClient.ListNotifications(ctx, request)
}

func main() {
	configs := config.LoadConfigs()

	// Internal gRPC listener (only reachable from inside the Docker network)
	lis, err := net.Listen("tcp", "api-gateway:"+configs.GATEWAY_PORT)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Dial user-service
	userConn, err := grpc.NewClient(
		"user-service:"+configs.USER_SERVICE_PORT,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Println("Failed to dial user-service:", err)
	}
	userClient := userpb.NewUserServiceClient(userConn)

	// Dial data-storage-service
	storageConn, err := grpc.NewClient(
		"data-storage-service:"+configs.STORAGE_PORT,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Println("Failed to dial data-storage-service:", err)
	}
	storageClient := storagepb.NewStorageServiceClient(storageConn)

	notificationConn, err := grpc.NewClient(
		"notification-service:"+configs.NOTIFICATION_PORT,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Println("Failed to dial notification-service:", err)
	}
	notificationClient := notificationpb.NewNotificationServiceClient(notificationConn)

	// Start internal gRPC server
	grpcServer := grpc.NewServer()

	pb.RegisterGatewayServiceServer(grpcServer, &server{
		userClient:         userClient,
		storageClient:      storageClient,
		notificationClient: notificationClient,
	})
	reflection.Register(grpcServer)
	go func() {
		log.Println(grpcServer.Serve(lis))
	}()

	// Dial the gateway's own gRPC server, so gwmux can forward REST → gRPC
	gwConn, err := grpc.NewClient(
		"api-gateway:"+configs.GATEWAY_PORT,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Println("Failed to dial gateway gRPC:", err)
	}

	gwmux := runtime.NewServeMux()
	if err := pb.RegisterGatewayServiceHandler(context.Background(), gwmux, gwConn); err != nil {
		log.Println("Failed to register gateway:", err)
	}

	// ONE public port: /api/* goes to gwmux, everything else to the file server
	root := http.NewServeMux()
	root.Handle("/api/", gwmux)
	root.Handle("/", http.FileServer(http.Dir("./frontend")))

	uploadService := service.NewFileUploadService(storageClient, userClient)
	uploadHandler := httpTransport.NewFileUploadHandler(uploadService)
	if err := uploadHandler.RegisterRoutes(gwmux); err != nil { //Register the upload route for files
		log.Printf("Failed to route to /api/file:%s", err)
	}

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: root,
	}

	log.Println("Serving frontend + API on http://0.0.0.0:8080")
	log.Fatalln(httpServer.ListenAndServe())
}
