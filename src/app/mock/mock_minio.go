package minio_mock

import (
	"github.com/minio/minio-go/v7"

	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
	*minio.Client
}

/*
func (m *MockClient) ListObjects(ctx context.Context, bucketName, prefix string, recursive bool) <-chan minio.ObjectInfo {
	ch := make(chan minio.ObjectInfo)
	go func() {
		defer close(ch)
		// Simulate a list of objects
		objects := []minio.ObjectInfo{
			{Key: "file1.txt"},
			{Key: "file2.jpg"},
			// Add more objects as needed for testing different scenarios
		}
		for _, obj := range objects {
			ch <- obj
		}
	}()
	return ch
}

func (m *MockClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	// Simulate a presigned URL
	return &url.URL{
		Scheme:   "https",
		Host:     "example.com",
		Path:     "/" + bucketName + "/" + objectName,
		RawQuery: reqParams.Encode(),
	}, nil
}

func (m *MockClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	// Simulate a successful object upload
	return 0, nil
}

func (m *MockClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	// Simulate a successful object removal
	return nil
}
*/
