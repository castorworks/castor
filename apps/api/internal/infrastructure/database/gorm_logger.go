package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/pkg/log"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// slowQueryThreshold 超过它的语句记一条 warn，便于发现缺索引等问题
const slowQueryThreshold = 200 * time.Millisecond

// gormLogger 把 GORM 的日志交给应用的 JSON 日志（zerolog）。GORM 自带的日志器输出带颜色控制码的
// 纯文本、把"查无记录"这种正常分支记成错误，并把参数值拼进 SQL——一条慢的
// UPDATE users SET password=… 就会把密码哈希写进日志。这里只记带占位符的 SQL（见 ParamsFilter）。
type gormLogger struct{}

var _ gormlogger.Interface = gormLogger{}
var _ gorm.ParamsFilter = gormLogger{}

func (l gormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

func (gormLogger) Info(ctx context.Context, msg string, args ...any) {
	log.InfoCtx(ctx).Str("component", "gorm").Msg(fmt.Sprintf(msg, args...))
}

func (gormLogger) Warn(ctx context.Context, msg string, args ...any) {
	log.WarnCtx(ctx).Str("component", "gorm").Msg(fmt.Sprintf(msg, args...))
}

func (gormLogger) Error(ctx context.Context, msg string, args ...any) {
	log.ErrCtx(ctx, fmt.Errorf(msg, args...)).Str("component", "gorm").Msg("Database error")
}

// ParamsFilter 让 GORM 交给 Trace 的 SQL 保留占位符，不代入参数值。
func (gormLogger) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		log.ErrCtx(ctx, err).Str("component", "gorm").Str("sql", sql).Int64("rows", rows).
			Dur("elapsed", elapsed).Msg("Database query failed")
	case elapsed > slowQueryThreshold:
		sql, rows := fc()
		log.WarnCtx(ctx).Str("component", "gorm").Str("sql", sql).Int64("rows", rows).
			Dur("elapsed", elapsed).Msg("Slow database query")
	}
}
