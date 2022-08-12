package problem

import (
	"context"
	"io"
	"sync"

	"rindag/service/storage"

	"github.com/go-git/go-billy/v5"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
)

// Bucket returns the name of the bucket where the problem is stored.
func (p *Problem) Bucket() (string, error) {
	bucket := p.ID.String()

	exist, err := storage.Client.BucketExists(context.Background(), bucket)
	if err != nil {
		return "", err
	}

	if !exist {
		if err := storage.Client.MakeBucket(
			context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return "", err
		}
	}

	return bucket, nil
}

func clearBucket(ctx context.Context, bucket string) error {
	objectsCh := make(chan minio.ObjectInfo, 16)

	for object := range storage.Client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Recursive: true,
	}) {
		if err := object.Err; err != nil {
			return err
		}
		objectsCh <- object
	}
	close(objectsCh)

	for err := range storage.Client.RemoveObjects(
		ctx, bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if err.Err != nil {
			return err.Err
		}
	}

	return nil
}

// StorageSave storages the problem in the storage provider.
func (p *Problem) StorageSave(testGroups map[string]*TestGroup, fs billy.Filesystem) error {
	bucket, err := p.Bucket()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clearBucket(ctx, bucket)

	errChan := make(chan error, 16)
	wg := &sync.WaitGroup{}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	copyToStorage := func(pa string) error {
		file, err := fs.Open(pa)
		if err != nil {
			return err
		}
		info, err := fs.Stat(pa)
		if info == nil {
			return err
		}

		log.WithField("size", info.Size()).Debug("Uploading file")

		if _, err := storage.Client.PutObject(
			ctx, bucket, pa, file, info.Size(), minio.PutObjectOptions{}); err != nil {
			return err
		}

		return nil
	}

	for _, group := range testGroups {
		wg.Add(len(group.Tests))
		for _, test := range group.Tests {
			go func(test TestCase) {
				infPath := test.Prefix + ".in"
				ansPath := test.Prefix + ".ans"

				if err := copyToStorage(infPath); err != nil {
					errChan <- err
					cancel()
					return
				}

				if err := copyToStorage(ansPath); err != nil {
					errChan <- err
					cancel()
					return
				}

				wg.Done()
			}(test)
		}
	}

	for err := range errChan {
		return err
	}
	return nil
}

// StorageLoad loads the problem from the storage provider, and save it to file system.
func (p *Problem) StorageLoad(testGroups map[string]*TestGroup, fs billy.Filesystem) error {
	bucket, err := p.Bucket()
	if err != nil {
		return err
	}

	ctx := context.Background()

	copyToFS := func(pa string) error {
		obj, err := storage.Client.GetObject(ctx, bucket, pa, minio.GetObjectOptions{})
		if err != nil {
			return err
		}

		file, err := fs.Create(pa)
		if err != nil {
			return err
		}

		if _, err := io.Copy(file, obj); err != nil {
			return err
		}

		return nil
	}

	for _, group := range testGroups {
		for _, test := range group.Tests {
			infPath := test.Prefix + ".in"
			ansPath := test.Prefix + ".ans"

			if err := copyToFS(infPath); err != nil {
				return err
			}
			if err := copyToFS(ansPath); err != nil {
				return err
			}
		}
	}

	return nil
}
