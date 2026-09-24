package persistence

import (
	"context"
	"fmt"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
)

// countByMonth 统计 table 在最近 months 个自然月（含本月）内按 created_at 分月的行数，
// 由数据库生成连续月份序列并补零，月份边界与 CURRENT_DATE 一样取数据库会话时区。
// table 与 condition 只能是代码中的常量，不得来自请求；opts 的条件（如数据范围）追加在
// 被统计的表上，同样由代码构造。
func countByMonth(ctx context.Context, db *gorm.DB, table, condition string, months int, opts ...query.Option) ([]shared.MonthlyCount, error) {
	if months < 1 {
		return []shared.MonthlyCount{}, nil
	}
	back := months - 1
	where := "created_at >= date_trunc('month', now()) - make_interval(months => ?)"
	args := []any{back, back}
	if condition != "" {
		where += " AND " + condition
	}
	for _, opt := range opts {
		where += " AND (" + opt.Condition + ")"
		args = append(args, opt.Args...)
	}
	sql := fmt.Sprintf(`SELECT to_char(m, 'YYYY-MM') AS month, COALESCE(c.count, 0) AS count
FROM generate_series(date_trunc('month', now()) - make_interval(months => ?), date_trunc('month', now()), interval '1 month') AS m
LEFT JOIN (SELECT date_trunc('month', created_at) AS month, count(*) AS count FROM %s WHERE %s GROUP BY 1) AS c ON c.month = m
ORDER BY m`, table, where)
	var rows []shared.MonthlyCount
	if err := db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("count %s by month: %w", table, err)
	}
	return rows, nil
}
