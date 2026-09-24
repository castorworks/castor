package department

import "context"

// Repository 部门仓储。写操作在 WithTx + Lock 内对整棵树校验后执行，
// 并发修改不会产生环或超深的树。
type Repository interface {
	WithTx(ctx context.Context, fn func(Repository) error) error
	Lock(ctx context.Context) error
	List(ctx context.Context) ([]Department, error)
	Save(ctx context.Context, d *Department) error
	Delete(ctx context.Context, id uint) error
}
