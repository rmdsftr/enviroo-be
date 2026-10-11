package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
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

const maxImageSize = 500 * 1024

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

func (cs *CloudflareStorage) UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folderName string) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file: %w", err)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ext := filepath.Ext(fileHeader.Filename)

	if strings.HasPrefix(contentType, "image/") && len(data) > maxImageSize {
		compressed, compErr := compressImage(data)
		if compErr == nil {
			data = compressed
			contentType = "image/jpeg"
			ext = ".jpg"
		}
	}

	newFileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	var objectKey string
	if folderName != "" {
		objectKey = fmt.Sprintf("%s/%s", folderName, newFileName)
	} else {
		objectKey = newFileName
	}

	_, err = cs.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(cs.config.CFBucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("gagal upload ke r2: %w", err)
	}

	publicURL := cs.config.CFPublicURL
	if !strings.HasSuffix(publicURL, "/") {
		publicURL += "/"
	}

	return publicURL + objectKey, nil
}

func compressImage(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gagal decode gambar: %w", err)
	}

	for quality := 80; quality >= 20; quality -= 10 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			continue
		}
		if buf.Len() <= maxImageSize {
			return buf.Bytes(), nil
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 20}); err != nil {
		return nil, fmt.Errorf("gagal encode gambar: %w", err)
	}
	return buf.Bytes(), nil
}
