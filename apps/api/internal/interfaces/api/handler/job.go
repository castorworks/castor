package handler

import (
	"fmt"
	"reflect"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AdminJobHandler 定时任务处理器
type AdminJobHandler struct {
	jobs  service.JobService
	audit service.AuditLogService
}

// NewAdminJobHandler 创建定时任务处理器
func NewAdminJobHandler(jobs service.JobService, audit service.AuditLogService) *AdminJobHandler {
	return &AdminJobHandler{jobs: jobs, audit: audit}
}

// List 列出代码里注册的定时任务及其设置、下次执行时间与最近一次执行
// GET /api/v1/admin/jobs
func (h *AdminJobHandler) List(c *gin.Context) {
	resp, err := h.jobs.List(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// Update 修改 cron 表达式与启停
// PUT /api/v1/admin/jobs/:key
func (h *AdminJobHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.JobUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	resp, err := h.jobs.Update(ucontext.WithAuditContext(c), key, req)
	logAudit(c, h.audit, audit_log.AuditLogTypeJobUpdate, key, fmt.Sprintf("Set cron %q, enabled=%t", req.Cron, req.IsEnabled), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, resp, response.InfJobUpdated)
}

// Run 立即在后台执行一次；返回刚创建的执行记录
// POST /api/v1/admin/jobs/:key/run
func (h *AdminJobHandler) Run(c *gin.Context) {
	key := c.Param("key")
	run, err := h.jobs.Trigger(c.Request.Context(), key, service.JobOperator{ID: ucontext.GetUserID(c), Username: ucontext.GetUsername(c)})
	details := "Started manual run"
	if run != nil {
		details = fmt.Sprintf("Started manual run #%d", run.ID)
	}
	logAudit(c, h.audit, audit_log.AuditLogTypeJobRun, key, details, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, run, response.InfJobTriggered)
}

// Runs 分页列出执行记录
// GET /api/v1/admin/job-runs
func (h *AdminJobHandler) Runs(c *gin.Context) {
	GenericGets(c, reflect.TypeOf(job.Run{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
		return h.jobs.ListRuns(ctx, page, size, order, opts...)
	})
}
