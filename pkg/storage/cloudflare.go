package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"enviroo-be/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type CloudflareStorage struct {
	client *s3.Client
	config *config.Config
}

func NewCloudflareStorage(cfg *config.Config) (*CloudflareStorage, error) {
	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.CFAccountID)

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               r2Endpoint,
			HostnameImmutable: true,
			SigningRegion:     region,
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.CFAccessKeyID, cfg.CFSecretAccessKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal meresolve aws config r2: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &CloudflareStorage{
		client: client,
		config: cfg,
	}, nil
}

// UploadFile menangani upload file foto atau video ke Cloudflare R2
func (cs *CloudflareStorage) UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folderName string) (string, error) {
	ext := filepath.Ext(fileHeader.Filename)
	// Randomize file name using uuid to prevent collisions
	newFileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Setup folder structure if provided
	var objectKey string
	if folderName != "" {
		objectKey = fmt.Sprintf("%s/%s", folderName, newFileName)
	} else {
		objectKey = newFileName
	}

	// Obtain ContentType
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := cs.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(cs.config.CFBucket),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("gagal upload ke r2: %w", err)
	}

	// Prepare public URL
	publicURL := cs.config.CFPublicURL
	if !strings.HasSuffix(publicURL, "/") {
		publicURL += "/"
	}

	return publicURL + objectKey, nil
}
