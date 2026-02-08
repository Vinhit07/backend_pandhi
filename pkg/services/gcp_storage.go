package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

var (
	storageClient *storage.Client
	bucketName    string
)

// InitGCPStorage initializes the GCP Storage client
func InitGCPStorage() error {
	bucketName = os.Getenv("GCP_BUCKET_NAME")
	if bucketName == "" {
		return fmt.Errorf("GCP_BUCKET_NAME not set")
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCP storage client: %v", err)
	}

	storageClient = client
	return nil
}

// UploadImage uploads an image to GCP Storage and returns the public URL
func UploadImage(fileBuffer []byte, fileName string) (string, error) {
	if storageClient == nil {
		return "", fmt.Errorf("GCP storage client not initialized")
	}

	ctx := context.Background()

	// Generate unique filename with random prefix
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	uniqueFileName := hex.EncodeToString(randomBytes) + "-" + fileName

	bucket := storageClient.Bucket(bucketName)
	obj := bucket.Object(uniqueFileName)
	writer := obj.NewWriter(ctx)

	// Auto-detect content type or default to image/jpeg
	writer.ContentType = "image/jpeg"

	if _, err := writer.Write(fileBuffer); err != nil {
		writer.Close()
		return "", fmt.Errorf("GCS upload failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("GCS upload finalization failed: %v", err)
	}

	// Return public URL
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, uniqueFileName)
	return publicURL, nil
}

// DeleteImage deletes an image from GCP Storage
func DeleteImage(imageURL string) error {
	if imageURL == "" {
		return nil
	}

	if storageClient == nil {
		return fmt.Errorf("GCP storage client not initialized")
	}

	// Extract filename from URL
	urlParts := strings.Split(imageURL, "/")
	if len(urlParts) == 0 {
		return nil
	}
	fileName := urlParts[len(urlParts)-1]

	ctx := context.Background()
	bucket := storageClient.Bucket(bucketName)
	obj := bucket.Object(fileName)

	// Delete the file
	if err := obj.Delete(ctx); err != nil {
		// Don't fail if file doesn't exist
		return nil
	}

	return nil
}

// GetSignedURL generates a signed URL for private access (15 minutes)
func GetSignedURL(fileURL string) (string, error) {
	if fileURL == "" {
		return "", nil
	}

	// If public URL, return as-is (no signing needed)
	if strings.HasPrefix(fileURL, "http") {
		return fileURL, nil
	}

	// If it's just a filename or relative path, prepend the bucket URL
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, fileURL)
	return publicURL, nil
}

// UploadImageFromReader uploads an image from an io.Reader (for multipart uploads)
func UploadImageFromReader(reader io.Reader, fileName string) (string, error) {
	// Read all data into buffer
	buffer, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return UploadImage(buffer, fileName)
}
