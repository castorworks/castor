package service

import (
	"context"
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
)

// dashboardRecentAuditLogs 仪表盘展示的最近审计日志条数
const dashboardRecentAuditLogs = 5

// DashboardService 仪表盘统计服务
//
// 用户、登录、会话与审计相关的数字都按调用者的数据范围统计；资产属于全组织共享的资产库，
// 统计保持全局。
type DashboardService interface {
	GetStats(ctx context.Context, scope permission.AccessScope) (*dto.DashboardStatsResp, error)
}

type dashboardService struct {
	users        user.Repository
	loginHistory login_history.Repository
	assetService AssetService
	auditLogs    AuditLogService
	sessions     SessionService
}

// NewDashboardService 创建仪表盘统计服务
func NewDashboardService(users user.Repository, loginHistory login_history.Repository, assetService AssetService, auditLogs AuditLogService, sessions SessionService) DashboardService {
	return &dashboardService{users: users, loginHistory: loginHistory, assetService: assetService, auditLogs: auditLogs, sessions: sessions}
}

func (svc *dashboardService) GetStats(ctx context.Context, scope permission.AccessScope) (*dto.DashboardStatsResp, error) {
	_, userCount, err := svc.users.Gets(ctx, 1, 1, "id desc", withUserScope(nil, scope, "id")...)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	todayLogins, err := svc.loginHistory.CountSuccessfulToday(ctx, withUserScope(nil, scope, "user_id")...)
	if err != nil {
		return nil, fmt.Errorf("count today logins: %w", err)
	}
	activeSessions, err := svc.sessions.CountActive(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("count active sessions: %w", err)
	}
	assetStats, err := svc.assetService.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("asset stats: %w", err)
	}
	trend, err := svc.monthlyTrend(ctx, scope)
	if err != nil {
		return nil, err
	}
	logs, _, err := svc.auditLogs.Gets(ctx, scope, 1, dashboardRecentAuditLogs, "id desc")
	if err != nil {
		return nil, fmt.Errorf("recent audit logs: %w", err)
	}
	recent := make([]dto.AuditLogResp, len(logs))
	for i := range logs {
		recent[i].FromEntity(&logs[i])
	}
	return &dto.DashboardStatsResp{
		UserCount:               userCount,
		TodayLoginCount:         todayLogins,
		ActiveSessionCount:      activeSessions,
		AssetCount:              assetStats.TotalCount,
		AssetTotalSize:          assetStats.TotalSize,
		AssetTotalSizeFormatted: assetStats.TotalSizeFormatted,
		AssetCategoryStats:      assetStats.CategoryStats,
		MonthlyTrend:            trend,
		RecentAuditLogs:         recent,
	}, nil
}

// monthlyTrend 合并两条按月序列。两者由数据库以同一方式生成（同一会话时区、同样的月数），
// 仍按月份键合并，而不是假设下标对齐。
func (svc *dashboardService) monthlyTrend(ctx context.Context, scope permission.AccessScope) ([]dto.DashboardMonthStat, error) {
	newUsers, err := svc.users.CountCreatedByMonth(ctx, dto.DashboardTrendMonths, withUserScope(nil, scope, "id")...)
	if err != nil {
		return nil, fmt.Errorf("count new users by month: %w", err)
	}
	logins, err := svc.loginHistory.CountSuccessfulByMonth(ctx, dto.DashboardTrendMonths, withUserScope(nil, scope, "user_id")...)
	if err != nil {
		return nil, fmt.Errorf("count logins by month: %w", err)
	}
	loginsByMonth := make(map[string]int64, len(logins))
	for _, m := range logins {
		loginsByMonth[m.Month] = m.Count
	}
	trend := make([]dto.DashboardMonthStat, len(newUsers))
	for i, m := range newUsers {
		trend[i] = dto.DashboardMonthStat{Month: m.Month, NewUsers: m.Count, Logins: loginsByMonth[m.Month]}
	}
	return trend, nil
}
