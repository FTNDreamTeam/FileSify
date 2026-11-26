package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	storagepb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrUnauthorized    = errors.New("invalid or expired token")
	ErrFileTooLarge    = errors.New("file exceeds the 100MB limit")
	ErrQuotaExceeded   = errors.New("remaining quota is less than file size")
	ErrUpstreamFailed  = errors.New("downstream service error")
	ErrInvalidFileType = errors.New("file type not allowed (whitelist rejection)")
	ErrCorruptedFile   = errors.New("file content does not match its extension")
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

	ext := strings.ToLower(filepath.Ext(input.Filename))
	allowedExtensions := map[string]bool{
		// Images
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true, ".bmp": true, ".tiff": true, ".ico": true,
		// Documents
		".pdf": true, ".txt": true, ".csv": true, ".md": true, ".json": true, ".xml": true, ".rtf": true,
		".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true, ".odt": true, ".ods": true,
		// Archives
		".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true,
		// Audio
		".mp3": true, ".wav": true, ".ogg": true, ".flac": true, ".m4a": true,
		// Video
		".mp4": true, ".mkv": true, ".webm": true, ".avi": true, ".mov": true,
	}
	if !allowedExtensions[ext] {
		return ErrInvalidFileType
	}

	headerBuffer := make([]byte, 512)
	n, err := input.FileContent.Read(headerBuffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	detectedMime := http.DetectContentType(headerBuffer[:n])

	allowedMimes := map[string]bool{
		// Images
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true, "image/bmp": true, "image/tiff": true, "image/x-icon": true, "image/svg+xml": true,
		// Documents
		"application/pdf": true, "text/plain": true, "text/csv": true, "text/markdown": true, "application/json": true, "text/xml": true, "application/xml": true, "application/rtf": true,
		"application/msword": true, "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"application/vnd.ms-powerpoint": true, "application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"application/vnd.oasis.opendocument.text": true, "application/vnd.oasis.opendocument.spreadsheet": true,
		// Archives
		"application/zip": true, "application/x-tar": true, "application/gzip": true, "application/x-7z-compressed": true, "application/vnd.rar": true, "application/x-rar-compressed": true,
		// Audio
		"audio/mpeg": true, "audio/wav": true, "audio/ogg": true, "audio/flac": true, "audio/x-m4a": true,
		// Video
		"video/mp4": true, "video/x-matroska": true, "video/webm": true, "video/x-msvideo": true, "video/quicktime": true,
		// Fallback for unknown but safe binaries
		"application/octet-stream": true,
	}
	if !allowedMimes[detectedMime] {
		return ErrCorruptedFile
	}

	input.FileContent = io.MultiReader(bytes.NewReader(headerBuffer[:n]), input.FileContent)

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
				MimeType:      detectedMime,
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

	hasher := sha256.New()
	teeReader := io.TeeReader(input.FileContent, hasher)

	buf := make([]byte, 32*1024) //32kb for the buffer
	for {
		n, readErr := teeReader.Read(buf) //Starts reading the file
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

	checksum := hex.EncodeToString(hasher.Sum(nil))
	if err := stream.Send(&storagepb.UploadFileRequest{
		Data: &storagepb.UploadFileRequest_Metadata{
			Metadata: &storagepb.FileMetaData{
				Checksum: checksum,
			},
		},
	}); err != nil {
		return fmt.Errorf("Couldn't generate checksum for file:%v", err)
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
	return int64(storageResp.GetQuota()) >= fileSize/mb, nil
}
