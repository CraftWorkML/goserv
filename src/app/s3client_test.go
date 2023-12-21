// app_test.go

package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"

	//mocking "goserv/src/app/mock"
	"github.com/stretchr/testify/assert"
)

// Embed the actual minio.Client interface
type MockMinioClient struct {
	mock.Mock
}

// Override the methods you want to mock
func (m *MockMinioClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	fmt.Println("print listObjects")
	args := m.Called(ctx, bucketName, opts)
	fmt.Println("after called")
	return args.Get(0).(chan minio.ObjectInfo)
}

func (m *MockMinioClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	args := m.Called(ctx, bucketName, objectName, expires, reqParams)
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *MockMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockMinioClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

func TestMinioS3Client(t *testing.T) {
	// Create a mock configuration
	mockMinioClient := new(MockMinioClient)
	mockConfig := &MinioS3Client{
		endpoint:        "mockEndpoint",
		accessKeyID:     "mockAccessKey",
		secretAccessKey: "mockSecretKey",
		useSSL:          true,
		bucketName:      "mockBucket",
		client:          mockMinioClient,
	}

	// Test ListObjects method

	t.Run("ListObjects", func(t *testing.T) {
		channel := make(chan minio.ObjectInfo, 1)
		channel <- minio.ObjectInfo{Key: "Mock"}
		close(channel)
		mockMinioClient.On(
			"ListObjects",
			mock.Anything,
			mock.Anything,
			mock.Anything).
			Return(channel)
		mockMinioClient.On(
			"PresignedGetObject",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything).
			Return(&url.URL{}, nil)
		objects, err := mockConfig.ListObjects("", nil)
		assert.NoError(t, err, "ListObjects() returned an error")
		assert.Len(t, objects, 1, "ListObjects() did not return the expected number of objects")
	})

	// Test UploadFile method
	t.Run("UploadFile", func(t *testing.T) {
		mockMinioClient.On(
			"PutObject",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything).Return(minio.UploadInfo{}, nil)

		fileContent := []byte("Hello, World!")
		reader := bytes.NewReader(fileContent)
		err := mockConfig.UploadFile("test.txt", reader, len(fileContent))
		assert.NoError(t, err, "UploadFile() returned an error")
	})

	// Test DeleteFile method
	t.Run("DeleteFile", func(t *testing.T) {
		mockMinioClient.On(
			"RemoveObject",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything).Return(nil)

		err := mockConfig.DeleteFile("test.txt")
		assert.NoError(t, err, "DeleteFile() returned an error")
	})

	// Test checkIn method
	t.Run("checkIn", func(t *testing.T) {
		key := "file.jpg"
		filters := []string{"jpg", "png", "gif"}
		result := checkIn(key, filters)
		assert.True(t, result, "checkIn() returned false, expected true")
	})
}
