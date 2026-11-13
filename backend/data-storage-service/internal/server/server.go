package server

import (
	"bytes"
	"context"
	"log"
	"strings"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
	pb "github.com/FTNDreamTeam/FileSify/backend/gen/data-storage-service-gen"
)

type StorageServer struct {
	pb.UnimplementedStorageServiceServer
	Store storage.ObjectStore
}

func NewStorageServer(store storage.ObjectStore) *StorageServer {
	return &StorageServer{Store: store}
}

func (s *StorageServer) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	folderPath := req.GetPath()
	if folderPath == "" {
		return &pb.CreateFolderResponse{Success: false, Message: "Path cannot be empty"}, nil
	}

	// Folders in Silo must end with a slash
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	// Create a 0-byte object
	emptyReader := bytes.NewReader([]byte{})

	err := s.Store.Put(ctx, "filesify", folderPath, emptyReader, 0, "application/x-directory")
	if err != nil {
		log.Printf("Create folder error: %v", err)
		return &pb.CreateFolderResponse{Success: false, Message: "Failed to create folder in Silo"}, nil
	}

	log.Printf("Successfully created folder: %s", folderPath)
	return &pb.CreateFolderResponse{Success: true, Message: "Folder created successfully"}, nil
}
