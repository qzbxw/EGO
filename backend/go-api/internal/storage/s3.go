// Package storage provides clients for interacting with external storage services, like S3.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"egobackend/internal/models"

	awsv1 "github.com/aws/aws-sdk-go/aws"
	credsv1 "github.com/aws/aws-sdk-go/aws/credentials"
	sessionv1 "github.com/aws/aws-sdk-go/aws/session"
	s3v1 "github.com/aws/aws-sdk-go/service/s3"
)

// S3Service provides methods for interacting with an S3-compatible object storage service.
type S3Service struct {
	client *s3v1.S3
	bucket string
}

// NewS3Service creates and configures a new S3Service instance.
// If the S3 configuration is incomplete, it returns a "null" service instance
// that will gracefully fail on operations, allowing the application to run
// without file storage functionality.
func NewS3Service(cfg models.S3Config) (*S3Service, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.KeyID == "" || cfg.AppKey == "" || cfg.Bucket == "" {
		log.Println("[S3] S3 configuration is not fully provided. File storage features will be disabled.")
		return &S3Service{client: nil, bucket: ""}, nil
	}

	disableSSL := strings.HasPrefix(strings.ToLower(cfg.Endpoint), "http://")

	sess, err := sessionv1.NewSession(&awsv1.Config{
		Region:           awsv1.String(cfg.Region),
		Endpoint:         awsv1.String(cfg.Endpoint),
		S3ForcePathStyle: awsv1.Bool(true),
		Credentials:      credsv1.NewStaticCredentials(cfg.KeyID, cfg.AppKey, ""),
		DisableSSL:       awsv1.Bool(disableSSL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	s3Client := s3v1.New(sess)

	log.Printf("[S3] (v1) S3 service initialized for bucket '%s' at endpoint '%s' (region '%s').", cfg.Bucket, cfg.Endpoint, cfg.Region)
	return &S3Service{client: s3Client, bucket: cfg.Bucket}, nil
}

// BucketName returns the name of the S3 bucket the service is configured for.
func (s *S3Service) BucketName() string {
	return s.bucket
}

// isConfigured checks if the S3 client is properly initialized.
func (s *S3Service) isConfigured() bool {
	return s.client != nil && s.bucket != ""
}

// UploadFile uploads a byte slice as an object to S3.
func (s *S3Service) UploadFile(ctx context.Context, key string, mimeType string, data []byte) error {
	return s.UploadStream(ctx, key, mimeType, bytes.NewReader(data))
}

// UploadStream uploads data from an io.Reader as an object to S3.
// This is more memory-efficient for large files as it avoids loading the entire file into memory.
func (s *S3Service) UploadStream(ctx context.Context, key string, mimeType string, r io.Reader) error {
	if !s.isConfigured() {
		return fmt.Errorf("S3 service is not configured; file upload is disabled")
	}

	// AWS SDK v1 PutObject expects an io.ReadSeeker for retries; ensure we have one
	var body io.ReadSeeker
	if rs, ok := r.(io.ReadSeeker); ok {
		body = rs
	} else {
		buf, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("failed to buffer upload body: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	_, err := s.client.PutObjectWithContext(ctx, &s3v1.PutObjectInput{
		Bucket:      awsv1.String(s.bucket),
		Key:         awsv1.String(key),
		Body:        body,
		ContentType: awsv1.String(mimeType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object '%s' to S3: %w", key, err)
	}
	log.Printf("[S3] Successfully uploaded '%s' to bucket '%s'.", key, s.bucket)
	return nil
}

// DeleteFiles deletes multiple objects from S3 in a single batch operation.
func (s *S3Service) DeleteFiles(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if !s.isConfigured() {
		log.Println("[S3] Skipping file deletion because S3 service is not configured.")
		return nil // Not a critical error.
	}

	objectsToDelete := make([]*s3v1.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objectsToDelete[i] = &s3v1.ObjectIdentifier{Key: awsv1.String(key)}
	}

	_, err := s.client.DeleteObjectsWithContext(ctx, &s3v1.DeleteObjectsInput{
		Bucket: awsv1.String(s.bucket),
		Delete: &s3v1.Delete{
			Objects: objectsToDelete,
			Quiet:   awsv1.Bool(true), // The response will only contain info about failed deletions.
		},
	})

	if err != nil {
		log.Printf("[S3] Error deleting objects from S3. Keys: %v, Error: %v", keys, err)
		return fmt.Errorf("failed to delete objects from S3: %w", err)
	}
	log.Printf("[S3] Successfully deleted %d object(s) from bucket '%s'.", len(keys), s.bucket)
	return nil
}

// DownloadFile downloads an S3 object and returns its content as a byte slice.
func (s *S3Service) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	rc, err := s.DownloadStream(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	body, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of object '%s' from S3: %w", key, err)
	}
	return body, nil
}

// DownloadStream returns the body of an S3 object as an io.ReadCloser.
// IMPORTANT: The caller is responsible for closing the returned ReadCloser.
func (s *S3Service) DownloadStream(ctx context.Context, key string) (io.ReadCloser, error) {
	if !s.isConfigured() {
		return nil, fmt.Errorf("S3 service is not configured; file download is disabled")
	}
	result, err := s.client.GetObjectWithContext(ctx, &s3v1.GetObjectInput{
		Bucket: awsv1.String(s.bucket),
		Key:    awsv1.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object '%s' from S3: %w", key, err)
	}
	return result.Body, nil
}
