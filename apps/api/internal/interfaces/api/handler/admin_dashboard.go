package handler

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/gin-gonic/gin"
)

// AdminDashboardHandler 管理员仪表盘处理器
type AdminDashboardHandler struct {
	userService         service.UserService
	assetService        service.AssetService
	loginHistoryService service.LoginHistoryService
	auditLogService     service.AuditLogService
}

// NewAdminDashboardHandler 创建管理员仪表盘处理器
func NewAdminDashboardHandler(
	userService service.UserService,
	assetService service.AssetService,
	loginHistoryService service.LoginHistoryService,
	auditLogService service.AuditLogService,
) *AdminDashboardHandler {
	return &AdminDashboardHandler{
		userService:         userService,
		assetService:        assetService,
		loginHistoryService: loginHistoryService,
		auditLogService:     auditLogService,
	}
}

// GetStats 获取仪表盘统计数据
// GET /api/v1/admin/dashboard/stats
func (h *AdminDashboardHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	// 获取用户总数
	_, userCount, err := h.userService.Gets(ctx, 1, 1, "id desc")
	if err != nil {
		log.Err(c, err).Msg("Failed to get user count")
		userCount = 0
	}

	// 获取资产统计
	assetStats, err := h.assetService.GetStats(ctx)
	if err != nil {
		log.Err(c, err).Msg("Failed to get asset stats")
	}

	// 获取今日登录次数（使用空查询获取总数）
	_, todayLoginCount, err := h.loginHistoryService.Gets(ctx, 1, 1, "id desc",
		*query.NewOption("created_at >= CURRENT_DATE"))
	if err != nil {
		log.Err(c, err).Msg("Failed to get today login count")
		todayLoginCount = 0
	}

	// 获取最近审计日志（5条）
	recentLogs, _, err := h.auditLogService.Gets(ctx, 1, 5, "id desc")
	if err != nil {
		log.Err(c, err).Msg("Failed to get recent audit logs")
		recentLogs = nil
	}

	result := gin.H{
		"userCount":       userCount,
		"todayLoginCount": todayLoginCount,
		"recentAuditLogs": recentLogs,
	}

	if assetStats != nil {
		result["assetCount"] = assetStats.TotalCount
		result["assetTotalSize"] = assetStats.TotalSize
		result["assetTotalSizeFormatted"] = assetStats.TotalSizeFormatted
		result["assetCategoryStats"] = assetStats.CategoryStats
	}

	response.Success(c, result)
}
