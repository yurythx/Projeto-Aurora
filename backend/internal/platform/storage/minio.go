package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioProvider implementa a interface Provider para interagir com o MinIO (ou AWS S3).
type MinioProvider struct {
	client *minio.Client
}

// NewMinioProvider constrói e conecta um cliente ao servidor MinIO.
func NewMinioProvider(endpoint, accessKey, secretKey string, useSSL bool) (*MinioProvider, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: failed to initialize minio client: %w", err)
	}

	return &MinioProvider{client: client}, nil
}

// EnsureBucket verifica se um bucket existe e cria caso não exista.
func (p *MinioProvider) EnsureBucket(ctx context.Context, bucketName string) error {
	exists, err := p.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("storage: check bucket exists: %w", err)
	}
	if !exists {
		err = p.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("storage: create bucket: %w", err)
		}
	}
	return nil
}

// Put salva um objeto (arquivo) no bucket.
func (p *MinioProvider) Put(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	_, err := p.client.PutObject(ctx, bucketName, objectName, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("storage: put object: %w", err)
	}
	return nil
}

// Get retorna um leitor para um objeto.
func (p *MinioProvider) Get(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	obj, err := p.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage: get object: %w", err)
	}
	return obj, nil
}

// Delete remove um objeto do bucket.
func (p *MinioProvider) Delete(ctx context.Context, bucketName, objectName string) error {
	err := p.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage: delete object: %w", err)
	}
	return nil
}

// PresignedPutURL gera uma URL segura temporária para que o cliente consiga fazer o upload via PUT sem passar pelo backend.
func (p *MinioProvider) PresignedPutURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	url, err := p.client.PresignedPutObject(ctx, bucketName, objectName, expiry)
	if err != nil {
		return "", fmt.Errorf("storage: generate presigned put: %w", err)
	}
	return url.String(), nil
}

// PresignedGetURL gera uma URL temporária para leitura.
func (p *MinioProvider) PresignedGetURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	url, err := p.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("storage: generate presigned get: %w", err)
	}
	return url.String(), nil
}
