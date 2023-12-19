package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ClientMinio interface {
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
}

type MinioS3Client struct {
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	useSSL          bool
	bucketName      string
	client          ClientMinio
}

const defaultContentType = "application/octet-stream"

// NewMinioS3Client creates a new MinioS3Client instance.
func NewMinioS3Client(endpoint, accessKeyID, secretAccessKey, bucketName string, useSSL bool) (*MinioS3Client, error) {

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Printf("can not create minio client %e with creds %s, %s, %s", err, endpoint, accessKeyID, secretAccessKey)
		return nil, fmt.Errorf("Failed to create Minio S3 client: %v", err)
	}

	return &MinioS3Client{
		endpoint:        endpoint,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		useSSL:          useSSL,
		bucketName:      bucketName,
		client:          minioClient,
	}, nil
}

func (s3 *MinioS3Client) ListObjects(prefix string, filters []string) ([]*url.URL, error) {
	ctx, cancel := context.WithCancel(context.Background())
	result := make([]*url.URL, 0)
	defer cancel()

	objectCh := s3.client.ListObjects(ctx, s3.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	fmt.Printf("print client list objects %v", objectCh)
	for object := range objectCh {
		if object.Err != nil {
			log.Printf("%v", object.Err)
			return result, object.Err
		}
		// Set request parameters for content-disposition.
		reqParams := make(url.Values)
		reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", object.Key))
		if len(filters) > 0 {
			if !checkIn(object.Key, filters) {
				continue
			}
		}
		// Generates a presigned url which expires in a day.
		presignedURL, err := s3.client.PresignedGetObject(context.Background(),
			s3.bucketName,
			object.Key,
			time.Second*24*60*60*7,
			reqParams)
		if err != nil {
			log.Printf("%e", err)
			return result, err
		}
		result = append(result, presignedURL)
	}
	return result, nil
}

// UploadFile uploads a file to the specified S3 bucket.
func (s3 *MinioS3Client) UploadFile(uploadPath string, object io.Reader, size int) error {
	_, err := s3.client.PutObject(context.Background(),
		s3.bucketName,
		uploadPath,
		object,
		int64(size),
		minio.PutObjectOptions{ContentType: defaultContentType})
	if err != nil {
		return fmt.Errorf("some error happened %v", err)
	}
	return nil
}

func (s3 *MinioS3Client) DeleteFile(fileName string) error {
	opts := minio.RemoveObjectOptions{}
	err := s3.client.RemoveObject(context.Background(), s3.bucketName, fileName, opts)
	log.Printf("remove %s, %s", s3.bucketName, fileName)
	if err != nil {
		log.Printf("%e", err)
		return fmt.Errorf("some error happened %v", err)
	}
	return nil
}

func checkIn(key string, filters []string) bool {
	parsed := strings.Split(key, ".")
	if len(parsed) > 0 {
		for _, f := range filters {
			if f == parsed[len(parsed)-1] {
				return true
			}
		}
	}
	return false
}
