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

type MinioS3Client struct {
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	useSSL          bool
	bucketName      string
	client          *minio.Client
}

// NewMinioS3Client creates a new MinioS3Client instance.
func NewMinioS3Client(endpoint, accessKeyID, secretAccessKey, bucketName string, useSSL bool) (*MinioS3Client, error) {

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("can not create minio client %e with creds %s, %s, %s", err, endpoint, accessKeyID, secretAccessKey)
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
	log.Printf("IMHERERE %s", prefix)
	objectCh := s3.client.ListObjects(ctx, s3.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	log.Printf("List got")
	for object := range objectCh {
		if object.Err != nil {
			log.Printf(fmt.Sprintf("%v", object.Err))
			return result, object.Err
		}
		log.Printf("Iterate over %v", object)
		fmt.Println(object)
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
			log.Printf(fmt.Sprintf("%e", err))
			return result, err
		}
		fmt.Println("Successfully generated presigned URL", presignedURL)
		result = append(result, presignedURL)
	}
	return result, nil
}

// UploadFile uploads a file to the specified S3 bucket.
func (s3 *MinioS3Client) UploadFile(uploadPath string, object io.Reader, size int) error {
	uploadInfo, err := s3.client.PutObject(context.Background(),
		s3.bucketName,
		uploadPath,
		object,
		int64(size),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("some error happened %v", err)
	}
	log.Printf("Successfully uploaded bytes: %v", uploadInfo)
	return nil
}

func (s3 *MinioS3Client) DeleteFile(fileName string) error {
	opts := minio.RemoveObjectOptions{}
	err := s3.client.RemoveObject(context.Background(), s3.bucketName, fileName, opts)
	log.Printf("remove %s, %s", s3.bucketName, fileName)
	if err != nil {
		log.Printf(fmt.Sprintf("%e", err))
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