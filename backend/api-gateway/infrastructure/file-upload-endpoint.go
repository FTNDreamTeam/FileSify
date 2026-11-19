package infrastructure

import (
	"fmt"
	storagepb "gen/data-storage-service-gen"
	userpb "gen/user-service-gen"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegisterFileUploadRoute(mux *runtime.ServeMux, storageClient storagepb.StorageServiceClient, userClient userpb.UserServiceClient) error {
	const kb = 1_000
	const mb = 1_000 * kb
	return mux.HandlePath("POST", "/api/storage/file", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		authResp, err := userClient.ValidateToken(r.Context(), &userpb.ValidateTokenRequest{Token: token}) //Returns UserID for selecting users root folder
		if err != nil {
			http.Error(w, "Invalid/expired token or user service unavailable", http.StatusUnauthorized)
			return
		}

		if authResp.GetUserId() == "" {
			http.Error(w, "Invalid/expired token or user service unavailable", http.StatusUnauthorized)
			return
		}

		if r.ContentLength > 100*mb {
			http.Error(w, "The file exceeds the 100MB limit", http.StatusBadRequest)
			return
		}

		if check, err := quotaLessThanLength(int32(r.ContentLength), userClient, authResp.UserId, r); err != nil {
			http.Error(w, "Error loading storage quota", http.StatusBadRequest)
			return
		} else if check {
			http.Error(w, "Remaining quota less than size of file", http.StatusBadRequest)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil { //Parses the file and keeps 10bytes in RAM the rest in disk temp folder
			http.Error(w, "Invalid multipart form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file") //Gets the file from request
		if err != nil {
			http.Error(w, "Missing 'file' field in form", http.StatusBadRequest)
			return
		}
		defer file.Close()

		folderPath := r.FormValue("path") //Gets the path we want to save the file in from the request

		stream, err := storageClient.UploadFile(r.Context()) //Opens a RPC stream to storageClient
		if err != nil {
			http.Error(w, "Failed to connect to storage service:"+err.Error(), http.StatusBadGateway)
			return
		}

		now := time.Now().UTC()
		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		err = stream.Send(&storagepb.UploadFileRequest{ //Sends metadata through the stream to storage client
			Data: &storagepb.UploadFileRequest_Metadata{
				Metadata: &storagepb.FileMetaData{
					Name:          header.Filename,
					MimeType:      mimeType,
					Path:          folderPath,
					UserId:        authResp.UserId,
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
			http.Error(w, "Failed to send file metadata", http.StatusBadGateway)
			return
		}

		buf := make([]byte, 32*1024) //32kb for the buffer
		for {
			n, readErr := file.Read(buf) //Starts reading the file
			if n > 0 {
				sendErr := stream.Send(&storagepb.UploadFileRequest{ //Sends part of file through RPC to storage client
					Data: &storagepb.UploadFileRequest_Content{
						buf[:n],
					},
				})
				if sendErr != nil {
					http.Error(w, "Failed while streaming file chuncks", http.StatusBadGateway)
					return
				}
			}
			if readErr == io.EOF { //EOF(End Of File)
				break
			}
			if readErr != nil {
				http.Error(w, "Error uploading file", http.StatusInternalServerError)
				return
			}
		}

		if _, err := stream.CloseAndRecv(); err != nil {
			http.Error(w, fmt.Sprintf("Storage service rejected upload: %v", err), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"uploaded"}`))
	})
}

func quotaLessThanLength(contentLength int32, userClient userpb.UserServiceClient, userId string, r *http.Request) (bool, error) {
	storageResp, err := userClient.CheckStorageQuota(r.Context(), &storagepb.CheckStorageQuotaRequest{
		UserId: userId,
	})
	if err != nil {
		return false, fmt.Errorf("Failed to fetch storage quota%v", err)
	}
	quota := storageResp.GetQuota()
	return quota < contentLength, nil
}
