package handler

import (
	"fmt"
	"strings"

	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// 审计约定：写接口在请求解析、参数校验通过之后（即真正尝试了操作），无论成败都恰好
// 记一条审计日志，成败取自操作返回的 err。只在成功分支记录会让越权尝试、冲突与失败
// 无迹可查；router_audit_test 据此禁止把字面量 nil 传给 err，也禁止绕过这里直接调用 LogAsync。

// logAudit 以当前登录用户为操作者记录一次写操作。
func logAudit(c *gin.Context, svc service.AuditLogService, logType audit_log.AuditLogType, target, details string, err error) {
	logAuditAs(c, svc, ucontext.GetUsername(c), ucontext.GetUserID(c), logType, target, details, err)
}

// logAuditAs 用于没有登录上下文的接口（找回密码、注册、外部登录回调），由调用方给出操作者。
func logAuditAs(c *gin.Context, svc service.AuditLogService, operator string, operatorID uint, logType audit_log.AuditLogType, target, details string, err error) {
	if svc == nil {
		return
	}
	if err != nil {
		details += " — failed: " + auditFailureReason(err)
	}
	svc.LogAsync(&audit_log.AuditLog{
		LogType:    logType,
		Operator:   operator,
		OperatorID: operatorID,
		Target:     target,
		Details:    details,
		IpAddr:     getClientIP(c),
		Success:    err == nil,
	})
}

// auditFailureReason 把失败原因收敛为已登记业务错误的固定文本；其余错误（SQL、Redis、
// 第三方返回等）可能带内部细节甚至敏感数据，一律记为 internal error。
func auditFailureReason(err error) string {
	if known := response.KnownError(err); known != nil {
		return known.Error()
	}
	return "internal error"
}

// maxAuditedKeys 限制批量操作写入审计详情的标识数量，避免单条详情无限增长。
const maxAuditedKeys = 20

// joinAuditKeys 把批量操作的标识（配置键、字典编码等）拼成一条有界的审计详情。
// 只用于非敏感标识：配置值、字典值、密码一类内容不得进入审计详情。
func joinAuditKeys(keys []string) string {
	if len(keys) <= maxAuditedKeys {
		return strings.Join(keys, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(keys[:maxAuditedKeys], ", "), len(keys)-maxAuditedKeys)
}

// joinAuditIDs 是 joinAuditKeys 的 ID 版本，用于批量删除一类按主键操作的接口。
func joinAuditIDs(ids []uint) string {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, fmt.Sprintf("%d", id))
	}
	return joinAuditKeys(keys)
}
