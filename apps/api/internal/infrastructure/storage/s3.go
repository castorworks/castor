package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/storage/s3"
)

// ensureBucketTimeout 启动时确认默认存储桶的时限
const ensureBucketTimeout = 15 * time.Second

// NewS3 创建 S3 存储客户端，并确认默认存储桶存在（不存在则创建）。
// gosuite v1 的 NewClient 不再发起网络请求；这里显式保留"存储不可用就启动失败"的行为，
// 否则问题要到第一次上传才暴露。
func NewS3() (*s3.Client, error) {
	client, err := s3.NewClient(&config.C.S3)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), ensureBucketTimeout)
	defer cancel()
	if err := client.EnsureBucket(ctx, ""); err != nil {
		return nil, fmt.Errorf("ensure s3 bucket %q: %w", config.C.S3.Bucket, err)
	}
	return client, nil
}
