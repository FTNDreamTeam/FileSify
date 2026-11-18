package server

import (
	"bytes"
	"context"
	"io"
	"log"
	"path"
	"strings"
	"time"

	pb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/db"
	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageServer struct {
	pb.UnimplementedStorageServiceServer
	Store      storage.ObjectStore
	UserClient userpb.UserServiceClient
	FileMeta   *mongo.Collection
	FolderMeta *mongo.Collection
}

func NewStorageServer(store storage.ObjectStore, userClient userpb.UserServiceClient, fileMeta *mongo.Collection, folderMeta *mongo.Collection) *StorageServer {
	return &StorageServer{Store: store, UserClient: userClient, FileMeta: fileMeta, FolderMeta: folderMeta}
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

	fileMeta := db.FolderMeta{
		ID:      primitive.NewObjectID(),
		Path:    folderPath,
		Created: primitive.NewDateTimeFromTime(time.Now()),
	}

	if req.GetToken() != "" { //Checks if request has token
		//If it does it means it is a user created folder
		valResponse, err := s.UserClient.ValidateToken(ctx, &userpb.ValidateTokenRequest{ //Validate token
			Token: req.GetToken(),
		})
		if err != nil {
			log.Printf("Error validating token:%v", err)
			return nil, status.Errorf(codes.Unauthenticated, "No token present:%v", err)
		}
		finalPath := valResponse.GetUserId() + "/" + folderPath // Folders in Silo must end with a slash
		folderPath = finalPath

		fileMeta.Owner, err = primitive.ObjectIDFromHex(valResponse.GetUserId())
		if err != nil {
			log.Printf("Couldn't create ObjectID from string:%v", err)
		}

	}
	//If it doesn't it means it is a root folder created after verification
	_, err := s.FolderMeta.InsertOne(ctx, fileMeta) //Insert metadata in Mongo

	if err != nil {
		log.Printf("Couldn't save metadata to MongoDB:%v", err)
	}

	// Create a 0-byte object
	emptyReader := bytes.NewReader([]byte{})

	err = s.Store.Put(ctx, "filesify", folderPath, emptyReader, 0, "application/x-directory")
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

	metaUserId, err := primitive.ObjectIDFromHex(meta.GetUserId())
	if err != nil {
		log.Printf("Couldn't parse ObjectID from string:%v", err)
	}

	_, err = s.FileMeta.InsertOne(stream.Context(), db.FileMeta{
		Name:  meta.GetName(),
		Date:  *meta.GetDateCreatedAt(),
		Time:  primitive.Timestamp{},
		Owner: metaUserId,
	})

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
