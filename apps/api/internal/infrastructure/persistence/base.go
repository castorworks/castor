package persistence

import (
	"context"
	"fmt"
	"sync"

	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var schemaCache sync.Map

// modelColumns returns the database column allowlist of model T, excluding sensitive columns.
func modelColumns[T any](db *gorm.DB) (func(string) bool, error) {
	s, err := schema.Parse(new(T), &schemaCache, db.NamingStrategy)
	if err != nil {
		return nil, err
	}
	return func(column string) bool {
		if query.IsSensitiveColumn(column) {
			return false
		}
		_, ok := s.FieldsByDBName[column]
		return ok
	}, nil
}

// safeOrder rebuilds a sort expression from the column allowlist of model T. Raw
// client strings never reach Order(): anything that is not `column [asc|desc]` on a
// real, non-sensitive column is rejected with query.ErrInvalidOrder.
func safeOrder[T any](db *gorm.DB, order string) (string, error) {
	allowed, err := modelColumns[T](db)
	if err != nil {
		return "", err
	}
	normalized, err := query.NormalizeOrder(order, allowed)
	if err != nil {
		return "", fmt.Errorf("%w: %q", err, order)
	}
	return normalized, nil
}

// PaginatedQuery 执行通用的分页查询
// T 是查询结果的模型类型
func PaginatedQuery[T any](
	ctx context.Context,
	db *gorm.DB,
	page, size int,
	order string,
	opts ...query.Option,
) ([]T, int64, error) {
	order, err := safeOrder[T](db, order)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 1
	}

	q := db.WithContext(ctx).Model(new(T))

	// 应用查询条件
	for _, opt := range opts {
		q = q.Where(opt.Condition, opt.Args...)
	}

	// 统计总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 应用排序
	if order != "" {
		q = q.Order(order)
	}

	// 执行分页查询
	var items []T
	if err := q.Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
