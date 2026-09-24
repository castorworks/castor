package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestGormLoggerSkipsRecordNotFoundAndFastQueries(t *testing.T) {
	calls := 0
	fc := func() (string, int64) { calls++; return "SELECT 1", 0 }
	l := gormLogger{}
	// "查无记录"是正常分支（例如首次 init-db 查找管理员），不应记成错误。
	l.Trace(context.Background(), time.Now(), fc, gorm.ErrRecordNotFound)
	l.Trace(context.Background(), time.Now(), fc, nil)
	if calls != 0 {
		t.Fatalf("record-not-found and fast queries must not be logged, SQL was built %d times", calls)
	}
	l.Trace(context.Background(), time.Now(), fc, errors.New("connection reset"))
	l.Trace(context.Background(), time.Now().Add(-time.Second), fc, nil)
	if calls != 2 {
		t.Fatalf("failed and slow queries must be logged, SQL was built %d times", calls)
	}
}

// 日志里的 SQL 只保留占位符：参数可能是密码哈希、联系方式、令牌。
func TestGormLoggerDropsParameterValues(t *testing.T) {
	sql, params := gormLogger{}.ParamsFilter(context.Background(), `UPDATE "users" SET "password"=$1`, "$2a$10$secret-hash")
	if sql != `UPDATE "users" SET "password"=$1` || params != nil {
		t.Fatalf("ParamsFilter = %q, %v", sql, params)
	}
}
