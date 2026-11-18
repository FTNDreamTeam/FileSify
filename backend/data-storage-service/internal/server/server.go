package server

import (
	"bytes"
	"context"
	"io"
	"log"
	"math"
	"path"
	"strings"
	"time"

	pb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"

	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/db"
	"github.com/FTNDreamTeam/FileSify/backend/data-storage-service/internal/storage"
	"go.mongodb.org/mongo-driver/bson"
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

	resp, err := s.UserClient.CheckStorageQuota(stream.Context(), &pb.CheckStorageQuotaRequest{
		UserId: meta.GetUserId(),
	})

	if err != nil {
		log.Printf("Error checking storage quota:%v", err)
		return status.Errorf(codes.Internal, "Error checking storage quota:%v", err)
	}

	remainingQuota := resp.GetQuota()

	if remainingQuota <= 0 {
		log.Printf("Remaining storage quota:%d", remainingQuota)
		return status.Error(codes.InvalidArgument, "There is no remaining storage quota")
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

	var totalBytes int32
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
			n, writeErr := pipeWriter.Write(chunk)
			if writeErr != nil {
				pipeWriter.CloseWithError(writeErr)
				return status.Errorf(codes.Internal, "Error writing to Silo stream: %v", writeErr)
			}
			totalBytes += int32(n)
		}
	}

	if err := <-uploadErrCh; err != nil {
		return status.Errorf(codes.Internal, "Failed to save file to Silo: %v", err)
	}

	megaBytes := int32(math.Ceil(float64(totalBytes) / (1024 * 1024)))

	var existingFile db.FileMeta
	err = s.FileMeta.FindOne(stream.Context(), bson.M{"path": objectKey}).Decode(&existingFile)
	if err != nil || err == mongo.ErrNoDocuments {
		log.Printf("Error looking for existing files in metadb:%v", err)
	}

	megaBytes = megaBytes - existingFile.Size

	if remainingQuota-megaBytes < 0 {
		err := s.Store.Delete(stream.Context(), "filesify", objectKey)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to delete object after checking remaining quota:%v", err)
		}
		return status.Error(codes.InvalidArgument, "The file is larger than the remaining storage quota")
	}
	if existingFile.Size >= 1 {
		_, err = s.FileMeta.UpdateByID(stream.Context(), existingFile.ID,
			bson.M{
				"$set": bson.M{
					"size": existingFile.Size + megaBytes,
					"date": *meta.GetDateCreatedAt(),
					"time": primitive.Timestamp{},
				},
			})
	} else {
		_, err = s.FileMeta.InsertOne(stream.Context(), db.FileMeta{
			Name:  meta.GetName(),
			Date:  *meta.GetDateCreatedAt(),
			Time:  primitive.Timestamp{},
			Owner: metaUserId,
			Path:  objectKey,
			Size:  megaBytes,
		})
	}

	_, err = s.UserClient.ChangeUsedQuota(stream.Context(), &pb.ChangeQuotaRequest{
		Amount: megaBytes,
		UserId: meta.GetUserId(),
	})

	if err != nil {
		return status.Errorf(codes.Internal, "Failed to decrease storage quota")
	}

	return stream.SendAndClose(&pb.UploadFileResponse{})
}
