package storage

import (
	"bytes"
	"context"
	"io"

	"rindag/service/etc"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
)

// MinIO is a storage backend for minio
type MinIO struct {
	client *minio.Client
	bucket string
}

// FromConfig creates a new MinIO storage backend from the given config
func FromConfig(cfg *etc.Configuration) (*MinIO, error) {
	if cfg.Storage.Type != "minio" {
		log.WithField("type", cfg.Storage.Type).Panic("Invalid storage type, expected minio")
	}
	client, err := minio.New(
		cfg.Storage.MinIO.Endpoint,
		&minio.Options{
			Creds: credentials.NewStaticV4(
				cfg.Storage.MinIO.AccessKeyID, cfg.Storage.MinIO.SecretAccessKey, ""),
			Secure: cfg.Storage.MinIO.UseSSL,
		},
	)
	if err != nil {
		log.WithError(err).Panic("Failed to initialize minio client")
	}
	log.Info("MinIO client initialized")
	return &MinIO{
		client: client,
		bucket: cfg.Storage.MinIO.Bucket,
	}, nil
}

// Bytes returns the bytes of the file
func (m *MinIO) Bytes(ctx context.Context, path string) ([]byte, error) {
	reader, err := m.client.GetObject(ctx, m.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func(reader *minio.Object) {
		err := reader.Close()
		if err != nil {
			log.WithError(err).Error("Failed to close minio reader")
		}
	}(reader)
	return io.ReadAll(reader)
}

// Write writes the object to minio
func (m *MinIO) Write(
	ctx context.Context,
	path string,
	data []byte,
) (minio.UploadInfo, error) {
	return m.client.PutObject(
		ctx,
		m.bucket,
		path,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{},
	)
}
