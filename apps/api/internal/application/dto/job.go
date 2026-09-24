package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/job"
)

// JobUpdateRequest 修改定时任务：只能调整 cron 表达式与启停，任务逻辑只在代码里注册。
type JobUpdateRequest struct {
	Cron      string `json:"cron" binding:"required,max=100"`
	IsEnabled bool   `json:"isEnabled"`
}

// JobResp 定时任务响应
type JobResp struct {
	job.Job
	DefaultCron    string     `json:"defaultCron"`    // 代码里注册的默认 cron
	TimeoutSeconds int64      `json:"timeoutSeconds"` // 单次执行超时
	NextRunAt      *time.Time `json:"nextRunAt"`      // 下次计划执行时间；停用时为空
	LastRun        *job.Run   `json:"lastRun"`        // 最近一次执行
}

// JobListResp 定时任务列表。cron 按 API 进程所在时区计算，TimeZone 供界面标注。
type JobListResp struct {
	TimeZone string    `json:"timeZone"`
	Jobs     []JobResp `json:"jobs"`
}
