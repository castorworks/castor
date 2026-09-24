package handler

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AdminDashboardHandler 管理员仪表盘处理器
type AdminDashboardHandler struct {
	dashboardService service.DashboardService
	rbac             service.RBACService
}

// NewAdminDashboardHandler 创建管理员仪表盘处理器
func NewAdminDashboardHandler(dashboardService service.DashboardService, rbac service.RBACService) *AdminDashboardHandler {
	return &AdminDashboardHandler{dashboardService: dashboardService, rbac: rbac}
}

// GetStats 获取仪表盘统计数据
// GET /api/v1/admin/dashboard/stats
func (h *AdminDashboardHandler) GetStats(c *gin.Context) {
	scope, ok := requestScope(c, h.rbac)
	if !ok {
		return
	}
	stats, err := h.dashboardService.GetStats(c.Request.Context(), scope)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, stats)
}
