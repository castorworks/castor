package job

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/pkg/query"
)

// Repository 定时任务设置与执行记录。
type Repository interface {
	// EnsureJobs 为代码里注册、数据库里还没有的任务插入默认设置；已有设置一律保留（只增不改）。
	EnsureJobs(ctx context.Context, defaults []Job) error
	ListJobs(ctx context.Context) ([]Job, error)
	GetJob(ctx context.Context, key string) (*Job, error)
	UpdateJob(ctx context.Context, item *Job) error

	CreateRun(ctx context.Context, run *Run) error
	FinishRun(ctx context.Context, id uint, status Status, affected int64, message string, finishedAt time.Time) error
	ListRuns(ctx context.Context, page, size int, order string, opts ...query.Option) ([]Run, int64, error)
	// LatestRuns 每个任务最近一次执行
	LatestRuns(ctx context.Context) (map[string]Run, error)
	// FailStaleRuns 把某任务在 before 之前开始、仍标为 RUNNING 的记录改为失败（执行它的进程已不在）；
	// 返回改动条数。
	FailStaleRuns(ctx context.Context, key, message string, before time.Time) (int64, error)
	// PruneRuns 每个任务只保留最近 keep 条执行记录
	PruneRuns(ctx context.Context, key string, keep int) error
}
