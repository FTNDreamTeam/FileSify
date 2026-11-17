package server

import (
	"bytes"
	"context"
	"io"
	"log"
	"path"
	"strings"

	pb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageServer struct {
	pb.UnimplementedStorageServiceServer
	Store      storage.ObjectStore
	UserClient userpb.UserServiceClient
}

func NewStorageServer(store storage.ObjectStore, userClient userpb.UserServiceClient) *StorageServer {
	return &StorageServer{Store: store, UserClient: userClient}
}

func (s *StorageServer) CreateFolder(ctx context.Context, req *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	folderPath := req.GetPath()
	if folderPath == "" {
		log.Print("Folder path emtpy")
		return nil, status.Error(codes.InvalidArgument, "path cannot be empty")
	}
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}
	if req.GetToken() != "" {
		valResponse, err := s.UserClient.ValidateToken(ctx, &userpb.ValidateTokenRequest{
			Token: req.GetToken(),
		})
		if err != nil {
			log.Printf("Error validating token:%v", err)
			return nil, status.Errorf(codes.Unauthenticated, "No token present:%v", err)
		}
		finalPath := valResponse.GetUserId() + "/" + folderPath
		folderPath = finalPath
	}

	// Folders in Silo must end with a slash

	// Create a 0-byte object
	emptyReader := bytes.NewReader([]byte{})

	err := s.Store.Put(ctx, "filesify", folderPath, emptyReader, 0, "application/x-directory")
	if err != nil {
		log.Printf("Create folder error: %v", err)
		return nil, status.Error(codes.Internal, "failed to create folder in Silo")
	}

	log.Printf("Successfully created folder: %s", folderPath)

	return &pb.CreateFolderResponse{}, nil
}

func (s *StorageServer) UploadFile(stream pb.StorageService_UploadFileServer) error {
	firstReq, err := stream.Recv() //Receive data through the stream
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Failed to receive metadata: %v", err)
	}

	meta := firstReq.GetMetadata() //Get the metadata from stream
	if meta == nil {
		return status.Errorf(codes.InvalidArgument, "First message must contain file metadata: %v", err)
	}

	safeUserID := path.Base(meta.GetUserId())                                     //Determine users root folder
	cleanSubFolder := strings.Trim(path.Clean("/"+meta.GetPath()), "/")           //Determine path to the file
	objectKey := path.Join(safeUserID, cleanSubFolder, path.Base(meta.GetName())) //Make object key

	contentType := meta.GetMimeType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	pipeReader, pipeWriter := io.Pipe() //Make reader and writer for stream and writing to object store
	uploadErrCh := make(chan error, 1)  //Error channel for eventual errors in read/write

	go func() {
		putErr := s.Store.Put( //Store the metadata in object store
			stream.Context(),
			"filesify",
			objectKey,
			pipeReader,
			-1,
			contentType,
		)
		uploadErrCh <- putErr
	}()

	for {
		req, err := stream.Recv() //Begin receiveing through stream
		if err == io.EOF {
			pipeWriter.Close()
			break
		}
		if err != nil {
			pipeWriter.CloseWithError(err)
			return status.Errorf(codes.Internal, "Error receiving chunk: %v", err)
		}

		chunk := req.GetContent() //Get part of file through stream
		if len(chunk) > 0 {
			if _, writeErr := pipeWriter.Write(chunk); writeErr != nil {
				pipeWriter.CloseWithError(writeErr)
				return status.Errorf(codes.Internal, "Error writing to Silo stream: %v", writeErr)
			}
		}
	}

	if err := <-uploadErrCh; err != nil {
		return status.Errorf(codes.Internal, "Failed to save file to Silo: %v", err)
	}

	return stream.SendAndClose(&pb.UploadFileResponse{})
}
