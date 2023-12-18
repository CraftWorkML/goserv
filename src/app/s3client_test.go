// app_test.go

package app

import (
	"bytes"
	"testing"

	"goserv/vendor/github.com/stretchr/testify/mock"

	"github.com/minio/minio-go/v7"

	//mocking "goserv/src/app/mock"
	"github.com/stretchr/testify/assert"
)

// Embed the actual minio.Client interface
type MockMinioClient struct {
	*minio.Client
	mock.Mock
}

/*
// Override the methods you want to mock
func (m *MockMinioClient) ListObjects(ctx context.Context, bucketName, prefix string, recursive bool) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucketName, prefix, recursive)
	return args.Get(0).(<-chan minio.ObjectInfo)
}

func (m *MockMinioClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	args := m.Called(ctx, bucketName, objectName, expires, reqParams)
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *MockMinioClient) PutObject(ctx context.Context, bucketName, objectName string, opts minio.PutObjectOptions) (n int64, err error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMinioClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}
*/
func TestMinioS3Client(t *testing.T) {
	// Create a mock configuration
	var temp minio.Client
	temp = MockMinioClient{}
	//mockMinio := new(MockMinioClient)
	mockConfig := &MinioS3Client{
		endpoint:        "mockEndpoint",
		accessKeyID:     "mockAccessKey",
		secretAccessKey: "mockSecretKey",
		useSSL:          true,
		bucketName:      "mockBucket",
		client:          &temp,
	}

	// Test ListObjects method
	t.Run("ListObjects", func(t *testing.T) {
		objects, err := mockConfig.ListObjects("", nil)
		assert.NoError(t, err, "ListObjects() returned an error")
		assert.Len(t, objects, 2, "ListObjects() did not return the expected number of objects")
	})

	// Test UploadFile method
	t.Run("UploadFile", func(t *testing.T) {
		fileContent := []byte("Hello, World!")
		reader := bytes.NewReader(fileContent)
		err := mockConfig.UploadFile("test.txt", reader, len(fileContent))
		assert.NoError(t, err, "UploadFile() returned an error")
	})

	// Test DeleteFile method
	t.Run("DeleteFile", func(t *testing.T) {
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
