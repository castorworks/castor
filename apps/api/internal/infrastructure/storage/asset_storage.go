package storage

import (
	"context"
	"io"

	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/hyperits/gosuite/storage/s3"
)

type assetStorage struct {
	client *s3.Client
}

// NewAssetStorage 把 S3 客户端适配为资产存储端口，所有对象写入配置的默认桶
func NewAssetStorage(client *s3.Client) asset.Storage {
	return &assetStorage{client: client}
}

func (s *assetStorage) Bucket() string { return s.client.Config().Bucket }

func (s *assetStorage) Put(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) error {
	return s.client.UploadObject(ctx, s.Bucket(), objectKey, r, size, contentType)
}

func (s *assetStorage) Get(ctx context.Context, objectKey string, w io.Writer) error {
	return s.client.GetObject(ctx, s.Bucket(), objectKey, w)
}

func (s *assetStorage) Delete(ctx context.Context, objectKey string) error {
	return s.client.DeleteObject(ctx, s.Bucket(), objectKey)
}
