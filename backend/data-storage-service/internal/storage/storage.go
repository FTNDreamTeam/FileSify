package storage

import (
	"context"
	"io"
)

type ObjectStore interface {
	//Put uploads an object to the specified bucket with the given name and content.
	Put(ctx context.Context, bucket, object string, r io.Reader, size int64, contentType string) error

	//Get retrieves an object from the specified bucket with the given name.
	Get(ctx context.Context, bucket, object string) (io.ReadCloser, error)

	//Delete removes an object from the specified bucket with the given name.
	Delete(ctx context.Context, bucket, object string) error

	//CreateBucket creates a new bucket with the given name.
	CreateBucket(ctx context.Context, bucket string) error
}
