package storage

import (
	"context"
	"io"
	"time"
)

// Provider define a interface de armazenamento de blobs/objetos (ex: PDFs).
type Provider interface {
	// Put salva um objeto (arquivo) no storage.
	Put(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) error

	// Get retorna o stream de leitura de um objeto salvo.
	Get(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)

	// Delete remove um objeto do storage.
	Delete(ctx context.Context, bucketName, objectName string) error

	// PresignedPutURL gera uma URL temporária para o frontend fazer upload diretamente.
	PresignedPutURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)

	// PresignedGetURL gera uma URL temporária para o frontend visualizar/fazer download diretamente.
	PresignedGetURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
}
