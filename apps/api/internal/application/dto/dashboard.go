package dto

import "github.com/castorworks/castor/internal/domain/asset"

// DashboardTrendMonths 是仪表盘趋势图覆盖的自然月数（含本月）。
const DashboardTrendMonths = 6

// DashboardMonthStat 仪表盘趋势中的一个自然月
type DashboardMonthStat struct {
	Month    string `json:"month"`    // 月份，形如 "2026-09"（数据库会话时区）
	NewUsers int64  `json:"newUsers"` // 当月新增用户数
	Logins   int64  `json:"logins"`   // 当月成功登录次数
}

// DashboardStatsResp 仪表盘统计响应。所有数字都在服务端聚合，不依赖前端拉取明细再计数。
type DashboardStatsResp struct {
	UserCount               int64                         `json:"userCount"`               // 用户总数
	TodayLoginCount         int64                         `json:"todayLoginCount"`         // 今日成功登录次数
	ActiveSessionCount      int64                         `json:"activeSessionCount"`      // 当前在线（未撤销、未过期）会话数
	AssetCount              int64                         `json:"assetCount"`              // 资产总数
	AssetTotalSize          int64                         `json:"assetTotalSize"`          // 资产总大小（字节）
	AssetTotalSizeFormatted string                        `json:"assetTotalSizeFormatted"` // 格式化的资产总大小
	AssetCategoryStats      map[asset.AssetCategory]int64 `json:"assetCategoryStats"`      // 资产分类统计
	MonthlyTrend            []DashboardMonthStat          `json:"monthlyTrend"`            // 最近几个月的新增用户与登录，按月份升序
	RecentAuditLogs         []AuditLogResp                `json:"recentAuditLogs"`         // 最近的审计日志
}
