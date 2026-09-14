package storage

import (
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/storage/s3"
)

// NewS3 创建 S3 存储客户端
func NewS3() (*s3.S3Client, error) {
	s3Client, err := s3.NewS3Client(&config.C.S3)
	if err != nil {
		return nil, err
	}
	return s3Client, nil
}
