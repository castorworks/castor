package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/query"
)

type trendUserRepo struct {
	*mockUserRepo
	months []shared.MonthlyCount
}

func (r *trendUserRepo) CountCreatedByMonth(context.Context, int, ...query.Option) ([]shared.MonthlyCount, error) {
	return r.months, nil
}

type trendLoginRepo struct {
	*mockLoginHistoryRepo
	months []shared.MonthlyCount
}

func (r *trendLoginRepo) CountSuccessfulByMonth(context.Context, int, ...query.Option) ([]shared.MonthlyCount, error) {
	return r.months, nil
}

// 两条序列按月份键合并：某个月只有登录、没有新用户（或反之）时不会错位。
func TestDashboardService_MonthlyTrendMergesByMonth(t *testing.T) {
	t.Parallel()
	svc := &dashboardService{
		users: &trendUserRepo{mockUserRepo: newMockUserRepo(), months: []shared.MonthlyCount{
			{Month: "2026-08", Count: 3}, {Month: "2026-09", Count: 0},
		}},
		loginHistory: &trendLoginRepo{months: []shared.MonthlyCount{
			{Month: "2026-09", Count: 12}, {Month: "2026-08", Count: 7},
		}},
	}
	got, err := svc.monthlyTrend(context.Background(), fullScope)
	if err != nil {
		t.Fatal(err)
	}
	want := []dto.DashboardMonthStat{
		{Month: "2026-08", NewUsers: 3, Logins: 7},
		{Month: "2026-09", NewUsers: 0, Logins: 12},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("monthlyTrend() = %+v, want %+v", got, want)
	}
}

// scopeCapturingRepos 记下仪表盘各项统计收到的条件。
type scopeCapturingUsers struct {
	*mockUserRepo
	gets, trend []query.Option
}

func (r *scopeCapturingUsers) Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]user.User, int64, error) {
	r.gets = opts
	return r.mockUserRepo.Gets(ctx, page, size, order, opts...)
}
func (r *scopeCapturingUsers) CountCreatedByMonth(_ context.Context, _ int, opts ...query.Option) ([]shared.MonthlyCount, error) {
	r.trend = opts
	return nil, nil
}

type scopeCapturingLogins struct {
	*mockLoginHistoryRepo
	today, trend []query.Option
}

func (r *scopeCapturingLogins) CountSuccessfulToday(_ context.Context, opts ...query.Option) (int64, error) {
	r.today = opts
	return 0, nil
}
func (r *scopeCapturingLogins) CountSuccessfulByMonth(_ context.Context, _ int, opts ...query.Option) ([]shared.MonthlyCount, error) {
	r.trend = opts
	return nil, nil
}

type scopeCapturingSessions struct {
	SessionService
	scope permission.AccessScope
}

func (s *scopeCapturingSessions) CountActive(_ context.Context, scope permission.AccessScope) (int64, error) {
	s.scope = scope
	return 0, nil
}

type scopeCapturingAudit struct {
	mockAuditLogService
	scope permission.AccessScope
}

func (a *scopeCapturingAudit) Gets(_ context.Context, scope permission.AccessScope, _, _ int, _ string, _ ...query.Option) ([]audit_log.AuditLog, int64, error) {
	a.scope = scope
	return nil, 0, nil
}

type staticAssetStats struct{ AssetService }

func (staticAssetStats) GetStats(context.Context) (*dto.AssetStatsResp, error) {
	return &dto.AssetStatsResp{}, nil
}

// 用户、登录、会话、审计的数字都按调用者数据范围统计。
func TestDashboardService_StatsFollowDataScope(t *testing.T) {
	t.Parallel()
	users := &scopeCapturingUsers{mockUserRepo: newMockUserRepo()}
	logins := &scopeCapturingLogins{mockLoginHistoryRepo: &mockLoginHistoryRepo{}}
	sessions := &scopeCapturingSessions{}
	audit := &scopeCapturingAudit{}
	svc := &dashboardService{users: users, loginHistory: logins, assetService: staticAssetStats{}, auditLogs: audit, sessions: sessions}
	scope := permission.AccessScope{UserID: 7, DepartmentIDs: []uint{2}}
	if _, err := svc.GetStats(context.Background(), scope); err != nil {
		t.Fatal(err)
	}
	userCondition := query.UserScope("id", false, 7, []uint{2}).Condition
	loginCondition := query.UserScope("user_id", false, 7, []uint{2}).Condition
	for name, opts := range map[string][]query.Option{"user count": users.gets, "new users trend": users.trend} {
		if len(opts) != 1 || opts[0].Condition != userCondition {
			t.Fatalf("%s not scoped: %+v", name, opts)
		}
	}
	for name, opts := range map[string][]query.Option{"today's logins": logins.today, "logins trend": logins.trend} {
		if len(opts) != 1 || opts[0].Condition != loginCondition {
			t.Fatalf("%s not scoped: %+v", name, opts)
		}
	}
	if sessions.scope.UserID != 7 || audit.scope.UserID != 7 || sessions.scope.All || audit.scope.All {
		t.Fatalf("sessions/audit must receive the caller's scope: %+v %+v", sessions.scope, audit.scope)
	}
}
