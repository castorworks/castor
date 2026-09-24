package job

import "time"

// Status 一次执行的状态；取值与字典 job_run_status 对应。
type Status string

const (
	StatusRunning   Status = "RUNNING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
)

// Trigger 一次执行的触发方式；取值与字典 job_trigger 对应。
type Trigger string

const (
	TriggerSchedule Trigger = "SCHEDULE"
	TriggerManual   Trigger = "MANUAL"
)

// Job 是一个定时任务的运营设置。任务本身（做什么）只在代码里注册，
// 数据库只保存可调整的部分：cron 表达式和启停；运营不能新增或改写任务逻辑。
type Job struct {
	Key       string    `json:"key"`
	Cron      string    `json:"cron"`
	IsEnabled bool      `json:"isEnabled"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy uint      `json:"updatedBy"`
}

// Run 是一次执行记录。Affected 是任务处理的条数；Message 只在失败时记录原因（技术细节，
// 已截断、不含敏感数据）。
type Run struct {
	ID         uint       `json:"id"`
	JobKey     string     `json:"jobKey"`
	Trigger    Trigger    `json:"trigger"`
	Status     Status     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	Affected   int64      `json:"affected"`
	Message    string     `json:"message"`
	Operator   string     `json:"operator"`
	OperatorID uint       `json:"operatorId"`
}
