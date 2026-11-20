package service

import (
	"context"
	"errors"
	"fmt"
	storagepb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"
	"io"
	"time"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrUnauthorized   = errors.New("invalid or expired token")
	ErrFileTooLarge   = errors.New("file exceeds the 100MB limit")
	ErrQuotaExceeded  = errors.New("remaining quota is less than file size")
	ErrUpstreamFailed = errors.New("downstream service error")
)

const kb = 1_000
const mb = 1_000 * kb
const maxUploadSize = 100 * mb

type UploadInput struct {
	Token       string
	Filename    string
	MimeType    string
	FolderPath  string
	FileSize    int64
	FileContent io.Reader
}

type FileUploadService struct {
	storageClient storagepb.StorageServiceClient
	userClient    userpb.UserServiceClient
}

func NewFileUploadService(storageClient storagepb.StorageServiceClient, userClient userpb.UserServiceClient) *FileUploadService {
	return &FileUploadService{
		storageClient: storageClient,
		userClient:    userClient,
	}
}

func (s *FileUploadService) Upload(context context.Context, input UploadInput) error {

	authResp, err := s.userClient.ValidateToken(context, &userpb.ValidateTokenRequest{Token: input.Token}) //Returns UserID for selecting users root folder
	if err != nil {
		return ErrUpstreamFailed
	}

	if authResp.GetUserId() == "" {
		return ErrUnauthorized
	}

	if input.FileSize > maxUploadSize {
		return ErrFileTooLarge
	}

	hasEnoughQuota, err := s.checkQuota(context, authResp.UserId, input.FileSize)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
	} else if !hasEnoughQuota {
		return ErrQuotaExceeded
	}

	stream, err := s.storageClient.UploadFile(context) //Opens a RPC stream to storageClient
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
	}

	now := time.Now().UTC()
	err = stream.Send(&storagepb.UploadFileRequest{ //Sends metadata through the stream to storage client
		Data: &storagepb.UploadFileRequest_Metadata{
			Metadata: &storagepb.FileMetaData{
				Name:          input.Filename,
				MimeType:      input.MimeType,
				Path:          input.FolderPath,
				UserId:        authResp.GetUserId(),
				TimeCreatedAt: timestamppb.New(now),
				DateCreatedAt: &date.Date{
					Year:  int32(now.Year()),
					Month: int32(now.Month()),
					Day:   int32(now.Day()),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("%w: Failed while streaming metadata: %v", ErrUpstreamFailed, err)
	}

	buf := make([]byte, 32*1024) //32kb for the buffer
	for {
		n, readErr := input.FileContent.Read(buf) //Starts reading the file
		if n > 0 {
			sendErr := stream.Send(&storagepb.UploadFileRequest{ //Sends part of file through RPC to storage client
				Data: &storagepb.UploadFileRequest_Content{
					buf[:n],
				},
			})
			if sendErr != nil {
				return fmt.Errorf("%w: Failed while streaming chunks: %v", ErrUpstreamFailed, err)
			}
		}
		if readErr == io.EOF { //EOF(End Of File)
			break
		}
		if readErr != nil {
			return fmt.Errorf(": Error reading file stream: %w", readErr)
		}
	}

	if _, err := stream.CloseAndRecv(); err != nil {
		return fmt.Errorf("%w: Storage service rejected upload: %v", ErrUpstreamFailed, err)
	}

	return nil
}

func (s *FileUploadService) checkQuota(ctx context.Context, userID string, fileSize int64) (bool, error) {
	storageResp, err := s.userClient.CheckStorageQuota(ctx, &storagepb.CheckStorageQuotaRequest{
		UserId: userID,
	})
	if err != nil {
		return false, err
	}
	return int64(storageResp.GetQuota()) >= fileSize, nil
}
