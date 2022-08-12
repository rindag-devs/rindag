package storage

import (
	"rindag/service/etc"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
)

// MinIO is a storage backend for minio
var Client *minio.Client

// FromConfig creates a new MinIO client from the given config
func FromConfig(cfg *etc.Configuration) (*minio.Client, error) {
	return minio.New(
		cfg.MinIO.Endpoint,
		&minio.Options{
			Creds: credentials.NewStaticV4(
				cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
			Secure: cfg.MinIO.UseSSL,
		},
	)
}

func init() {
	var err error
	Client, err = FromConfig(etc.Config)
	if err != nil {
		log.WithError(err).Fatal("Failed to create MinIO client")
	}
}
