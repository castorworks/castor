package handler

import (
	"fmt"
	"reflect"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// LoginHistoryHandler 用户登录历史处理器（用户查询自己的登录记录）
type LoginHistoryHandler struct {
	LoginHistoryService service.LoginHistoryService
}

// NewLoginHistoryHandler 创建用户登录历史处理器
func NewLoginHistoryHandler(svc service.LoginHistoryService) *LoginHistoryHandler {
	return &LoginHistoryHandler{LoginHistoryService: svc}
}

// Gets 用户查询自己的登录历史
func (h *LoginHistoryHandler) Gets(c *gin.Context) {
	username := ucontext.GetUsername(c)
	if username == "" {
		return
	}

	GenericGets(c, reflect.TypeOf(login_history.LoginHistory{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		// 强制追加 username 过滤条件
		usernameOpt := *query.NewOption("username = ?", username)
		opts = append(opts, usernameOpt)
		// 本人的记录由用户名条件限定，不再叠加管理端的数据范围。
		return h.LoginHistoryService.Gets(ctx, permission.AccessScope{All: true}, page, size, order, opts...)
	})
}

// AdminLoginHistoryHandler 管理员登录历史处理器
type AdminLoginHistoryHandler struct {
	LoginHistoryService service.LoginHistoryService
	AuditLogService     service.AuditLogService
	RBAC                service.RBACService
	Dictionary          service.DictionaryService
}

// NewAdminLoginHistoryHandler 创建管理员登录历史处理器
func NewAdminLoginHistoryHandler(svc service.LoginHistoryService, auditLogService service.AuditLogService, rbac service.RBACService, dictionary service.DictionaryService) *AdminLoginHistoryHandler {
	return &AdminLoginHistoryHandler{
		LoginHistoryService: svc,
		AuditLogService:     auditLogService,
		RBAC:                rbac,
		Dictionary:          dictionary,
	}
}

// Gets 获取登录历史列表
func (h *AdminLoginHistoryHandler) Gets(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericGets(c, reflect.TypeOf(login_history.LoginHistory{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.LoginHistoryService.Gets(ctx, scope, page, size, order, opts...)
	})
}

// Delete 删除登录历史
func (h *AdminLoginHistoryHandler) Delete(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericDelete(c, func(ctx *gin.Context, id uint) error {
		err := h.LoginHistoryService.Delete(ctx, scope, id)
		// 删除登录痕迹本身是高风险操作，必须留痕。
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeLoginHistoryDelete, fmt.Sprintf("%d", id),
			fmt.Sprintf("Delete login history ID: %d", id), err)
		return err
	})
}

// BatchDelete 批量删除登录历史
func (h *AdminLoginHistoryHandler) BatchDelete(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	GenericBatchDelete(c, &dto.LoginHistoryBatchDeleteReq{}, func(ctx *gin.Context, req interface{}) error {
		ids := req.(*dto.LoginHistoryBatchDeleteReq).Ids
		err := h.LoginHistoryService.BatchDelete(ctx, scope, ids)
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeLoginHistoryDelete, joinAuditIDs(ids),
			fmt.Sprintf("Batch delete %d login history records: %s", len(ids), joinAuditIDs(ids)), err)
		return err
	}, func(req interface{}) bool {
		return len(req.(*dto.LoginHistoryBatchDeleteReq).Ids) > 0
	})
}

// Export 按登录历史列表相同的筛选与数据范围导出
// GET /api/v1/admin/login-histories/export
func (h *AdminLoginHistoryHandler) Export(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	x := newExportContext(c, h.Dictionary)
	columns := []exportColumn[dto.LoginHistoryResp]{
		{"ColumnTime", func(l dto.LoginHistoryResp) string { return x.time(l.CreatedAt) }},
		{"ColumnUsername", func(l dto.LoginHistoryResp) string { return l.Username }},
		{"ColumnLoginMethod", func(l dto.LoginHistoryResp) string { return x.label("login_method", l.LoginMethod) }},
		{"ColumnResult", func(l dto.LoginHistoryResp) string { return x.result(l.Success) }},
		{"ColumnIpAddr", func(l dto.LoginHistoryResp) string { return l.IpAddr }},
		{"ColumnUserAgent", func(l dto.LoginHistoryResp) string { return l.UserAgent }},
	}
	exportList(c, reflect.TypeOf(login_history.LoginHistory{}), "login-histories", columns, func(ctx *gin.Context, page, size int, order string, opts ...query.Option) ([]dto.LoginHistoryResp, int64, error) {
		return h.LoginHistoryService.Gets(ctx, scope, page, size, order, opts...)
	}, func(rows int, err error) {
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeLoginHistoryExport, "-", fmt.Sprintf("Export %d login histories", rows), err)
	})
}
