package asset

import (
	"context"
	"io"
)

// Storage 资产文件本体的存储端口，由 infrastructure/storage 适配到对象存储。
type Storage interface {
	// Bucket 返回文件写入的存储桶
	Bucket() string

	// Put 写入对象
	Put(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) error

	// Get 把对象内容写入 w
	Get(ctx context.Context, objectKey string, w io.Writer) error

	// Delete 删除对象
	Delete(ctx context.Context, objectKey string) error
}
