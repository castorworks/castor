package database

import (
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/db/postgres"
	"gorm.io/gorm"
)

// NewPostgres 仅创建连接；结构和种子数据由 init-db 命令初始化。
func NewPostgres() (*gorm.DB, error) {
	client, err := postgres.NewClient(&config.C.Postgres)
	if err != nil {
		return nil, err
	}

	return client.DB(), nil
}

// Close releases the connection pool behind db.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
