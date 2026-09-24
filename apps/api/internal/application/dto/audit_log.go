package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/audit_log"
)

//
// Request DTOs
//

// AuditLogDeleteBeforeReq 按时间清理审计日志请求
type AuditLogDeleteBeforeReq struct {
	// RetentionDays 保留天数，删除该天数之前的日志
	RetentionDays int `json:"retentionDays" binding:"required,min=1"`
}

//
// Response DTOs
//

// AuditLogResp 审计日志响应
type AuditLogResp struct {
	ID         uint                   `json:"id"`
	CreatedAt  time.Time              `json:"createdAt"`
	LogType    audit_log.AuditLogType `json:"logType"`
	Operator   string                 `json:"operator"`
	OperatorID uint                   `json:"operatorId"`
	Target     string                 `json:"target"`
	Details    string                 `json:"details"`
	IpAddr     string                 `json:"ipAddr"`
	Success    bool                   `json:"success"`
}

// FromEntity 从审计日志实体转换
func (r *AuditLogResp) FromEntity(entity *audit_log.AuditLog) {
	r.ID = entity.ID
	r.CreatedAt = entity.CreatedAt
	r.LogType = entity.LogType
	r.Operator = entity.Operator
	r.OperatorID = entity.OperatorID
	r.Target = entity.Target
	r.Details = entity.Details
	r.IpAddr = entity.IpAddr
	r.Success = entity.Success
}

// AuditLogDeleteBeforeResp 清理审计日志响应
type AuditLogDeleteBeforeResp struct {
	Deleted int64 `json:"deleted"`
}
