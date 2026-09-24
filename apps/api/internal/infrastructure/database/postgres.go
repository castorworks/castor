package database

import (
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/hyperits/gosuite/db/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// NewPostgres 仅创建连接；结构和种子数据由 init-db 命令初始化。
func NewPostgres() (*gorm.DB, error) {
	conf := config.C.Postgres
	// 与 gosuite 的默认配置一样用单数表名，另把日志交给应用的 JSON 日志（见 gormLogger）。
	conf.GormConfig = &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}, Logger: gormLogger{}}
	client, err := postgres.NewClient(&conf)
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
