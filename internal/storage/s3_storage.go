package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Storage struct {
	client  *s3.Client
	bucket  string
	baseURL string
}

type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileURL   string `json:"file_url"`
	Key       string `json:"key"`
}

func NewS3Storage(region, bucket, accessKeyID, secretAccessKey, baseURL string) *S3Storage {
	var cfg aws.Config
	var err error

	// If credentials are provided, use them. Otherwise, use default credential chain
	if accessKeyID != "" && secretAccessKey != "" {
		cfg = aws.Config{
			Region: region,
			Credentials: credentials.NewStaticCredentialsProvider(
				accessKeyID,
				secretAccessKey,
				"",
			),
		}
	} else {
		// Use default credential chain (environment variables, ~/.aws/credentials, IAM role, etc.)
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
		)
		if err != nil {
			// If default config fails, create a basic config with region only
			cfg = aws.Config{
				Region: region,
			}
		}
	}

	client := s3.NewFromConfig(cfg)

	return &S3Storage{
		client:  client,
		bucket:  bucket,
		baseURL: baseURL,
	}
}

// GeneratePresignedURL generates a pre-signed URL for uploading a file
// Deprecated: Use GeneratePresignedURLWithFolder instead
func (s *S3Storage) GeneratePresignedURL(filename, contentType string) (*PresignedURLResponse, error) {
	return s.GeneratePresignedURLWithFolder(filename, contentType, "uploads")
}

// GeneratePresignedURLWithFolder generates a pre-signed URL for uploading a file to a specific folder
func (s *S3Storage) GeneratePresignedURLWithFolder(filename, contentType, folder string) (*PresignedURLResponse, error) {
	// Generate unique key
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	// Create presign client
	presignClient := s3.NewPresignClient(s.client)

	// Generate presigned PUT URL (valid for 15 minutes)
	presignedReq, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Generate final file URL
	var fileURL string
	if s.baseURL != "" {
		// Use CloudFront or custom domain
		fileURL = fmt.Sprintf("%s/%s", s.baseURL, key)
	} else {
		// Use S3 direct URL
		fileURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.client.Options().Region, key)
	}

	return &PresignedURLResponse{
		UploadURL: presignedReq.URL,
		FileURL:   fileURL,
		Key:       key,
	}, nil
}

// ValidateFileSize validates the file size
func (s *S3Storage) ValidateFileSize(size int64, maxSize int64) error {
	if size > maxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxSize)
	}
	return nil
}

// ValidateContentType validates the content type
func (s *S3Storage) ValidateContentType(contentType string, allowedTypes []string) error {
	for _, allowed := range allowedTypes {
		if contentType == allowed {
			return nil
		}
	}
	return fmt.Errorf("content type %s is not allowed", contentType)
}
