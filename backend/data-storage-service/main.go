package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// Endpoint: "localhost:9000" when running on host,
	// "silo:9000" when running inside the compose network.
	endpoint := os.Getenv("SILO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ROOT_USER")
	secretKey := os.Getenv("MINIO_ROOT_PASSWORD")
	if accessKey == "" || secretKey == "" {
		log.Fatal("MINIO_ROOT_USER and MINIO_ROOT_PASSWORD must be set")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln("Silo client init failed:", err)
	}
	log.Println("Connected to Silo at", endpoint)

	ctx := context.Background()
	bucketName := "filesify"

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatalln("Bucket check failed:", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			log.Fatalln("Bucket creation failed:", err)
		}
		log.Println("Bucket created:", bucketName)
	} else {
		log.Println("Bucket already exists:", bucketName)
	}

	// Smoke test: upload a small object
	objectName := "hello.txt"
	content := []byte("Hello from Go to Silo!\n")
	info, err := client.PutObject(
		ctx, bucketName, objectName,
		bytes.NewReader(content), int64(len(content)),
		minio.PutObjectOptions{ContentType: "text/plain"},
	)
	if err != nil {
		log.Fatalln("Upload failed:", err)
	}
	fmt.Printf("Uploaded %s (%d bytes)\n", info.Key, info.Size)

	// Verify by reading it back
	obj, err := client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln("GetObject failed:", err)
	}
	defer obj.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj); err != nil {
		log.Fatalln("Read failed:", err)
	}
	fmt.Printf("Read back: %q\n", buf.String())
}
