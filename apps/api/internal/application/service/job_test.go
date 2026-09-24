package service

import (
	"context"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/redis/go-redis/v9"
)

// memJobRepo 是 job.Repository 的内存实现
type memJobRepo struct {
	mu     sync.Mutex
	jobs   map[string]job.Job
	runs   []job.Run
	nextID uint
}

func newMemJobRepo() *memJobRepo { return &memJobRepo{jobs: map[string]job.Job{}} }

func (r *memJobRepo) EnsureJobs(_ context.Context, defaults []job.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range defaults {
		if _, ok := r.jobs[d.Key]; !ok {
			r.jobs[d.Key] = d
		}
	}
	return nil
}

func (r *memJobRepo) ListJobs(context.Context) ([]job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]job.Job, 0, len(r.jobs))
	for _, item := range r.jobs {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (r *memJobRepo) GetJob(_ context.Context, key string) (*job.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.jobs[key]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return &item, nil
}

func (r *memJobRepo) UpdateJob(_ context.Context, item *job.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[item.Key] = *item
	return nil
}

func (r *memJobRepo) CreateRun(_ context.Context, run *job.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	run.ID = r.nextID
	r.runs = append(r.runs, *run)
	return nil
}

func (r *memJobRepo) FinishRun(_ context.Context, id uint, status job.Status, affected int64, message string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.runs {
		if r.runs[i].ID == id {
			r.runs[i].Status, r.runs[i].Affected, r.runs[i].Message, r.runs[i].FinishedAt = status, affected, message, &at
		}
	}
	return nil
}

func (r *memJobRepo) ListRuns(context.Context, int, int, string, ...query.Option) ([]job.Run, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]job.Run(nil), r.runs...), int64(len(r.runs)), nil
}

func (r *memJobRepo) LatestRuns(context.Context) (map[string]job.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]job.Run{}
	for _, run := range r.runs {
		out[run.JobKey] = run
	}
	return out, nil
}

func (r *memJobRepo) FailStaleRuns(_ context.Context, key, message string, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for i := range r.runs {
		if r.runs[i].JobKey == key && r.runs[i].Status == job.StatusRunning && r.runs[i].StartedAt.Before(before) {
			r.runs[i].Status, r.runs[i].Message, r.runs[i].FinishedAt = job.StatusFailed, message, &before
			n++
		}
	}
	return n, nil
}

func (r *memJobRepo) PruneRuns(context.Context, string, int) error { return nil }

func (r *memJobRepo) run(id uint) job.Run {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, run := range r.runs {
		if run.ID == id {
			return run
		}
	}
	return job.Run{}
}

func (r *memJobRepo) runCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.runs)
}

// waitRun 等待执行记录离开 RUNNING
func (r *memJobRepo) waitRun(t *testing.T, id uint) job.Run {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if run := r.run(id); run.Status != job.StatusRunning {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("run %d did not finish", id)
	return job.Run{}
}

func newJobTestService(t *testing.T, repo *memJobRepo, rdb redis.UniversalClient, catalog JobCatalog) *jobService {
	t.Helper()
	svc := NewJobService(repo, catalog, rdb, rediskey.Namespace("test")).(*jobService)
	t.Cleanup(svc.Stop)
	return svc
}

func newJobTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestParseCron(t *testing.T) {
	spec, _, err := ParseCron("  0   3 * *  1-5 ")
	if err != nil || spec != "0 3 * * 1-5" {
		t.Fatalf("ParseCron = %q, %v; want normalized spec", spec, err)
	}
	for _, bad := range []string{"", "* * * *", "0 0 * * * *", "@hourly", "@every 5s", "TZ=UTC 0 3 * * *", "CRON_TZ=UTC 0 3 * * *", "61 * * * *", "a b c d e", "0 0 30 2 *"} {
		if _, _, err := ParseCron(bad); !errors.Is(err, apperror.ErrInvalidCron) {
			t.Errorf("ParseCron(%q) err = %v; want ErrInvalidCron", bad, err)
		}
	}
}

func TestJobCatalog_SweepRegisteredOnlyWhenEnabled(t *testing.T) {
	keys := func(c JobCatalog) []string {
		var out []string
		for _, d := range c {
			out = append(out, d.Key)
			if _, _, err := ParseCron(d.DefaultCron); err != nil || d.Timeout <= 0 || d.Run == nil {
				t.Errorf("job %s has an invalid definition", d.Key)
			}
		}
		return out
	}
	if got := keys(NewJobCatalog(nil, AssetSweepPolicy{}, nil, nil)); len(got) != 1 || got[0] != JobPurgeExpiredSessions {
		t.Fatalf("catalog without sweep = %v", got)
	}
	if got := keys(NewJobCatalog(nil, AssetSweepPolicy{OrphanTTL: time.Hour}, nil, nil)); len(got) != 2 {
		t.Fatalf("catalog with sweep = %v", got)
	}
}

func TestJobService_StartListAndUpdate(t *testing.T) {
	repo := newMemJobRepo()
	// 库里残留一个代码里已下线的任务：不应出现在列表里
	repo.jobs["retired"] = job.Job{Key: "retired", Cron: "* * * * *", IsEnabled: true}
	repo.jobs["b"] = job.Job{Key: "b", Cron: "5 4 * * *", IsEnabled: false}
	_, rdb := newJobTestRedis(t)
	noop := func(context.Context) (int64, error) { return 0, nil }
	svc := newJobTestService(t, repo, rdb, JobCatalog{
		{Key: "a", DefaultCron: "0 * * * *", DefaultEnabled: true, Timeout: time.Minute, Run: noop},
		{Key: "b", DefaultCron: "0 3 * * *", DefaultEnabled: true, Timeout: time.Minute, Run: noop},
	})
	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Jobs) != 2 || list.Jobs[0].Key != "a" || list.Jobs[1].Key != "b" || list.TimeZone == "" {
		t.Fatalf("List = %+v", list)
	}
	if list.Jobs[0].NextRunAt == nil || list.Jobs[0].DefaultCron != "0 * * * *" || list.Jobs[0].TimeoutSeconds != 60 {
		t.Fatalf("job a = %+v", list.Jobs[0])
	}
	// 已有设置保留，不被默认值覆盖；停用的任务没有下次执行时间
	if b := list.Jobs[1]; b.Cron != "5 4 * * *" || b.IsEnabled || b.NextRunAt != nil {
		t.Fatalf("job b = %+v", b)
	}

	if _, err := svc.Update(ctx, "a", dto.JobUpdateRequest{Cron: "every minute"}); !errors.Is(err, apperror.ErrInvalidCron) {
		t.Fatalf("Update invalid cron err = %v", err)
	}
	if _, err := svc.Update(ctx, "retired", dto.JobUpdateRequest{Cron: "* * * * *"}); !errors.Is(err, apperror.ErrJobNotFound) {
		t.Fatalf("Update unregistered job err = %v", err)
	}
	resp, err := svc.Update(ctx, "a", dto.JobUpdateRequest{Cron: "*/5  * * * *", IsEnabled: false})
	if err != nil || resp.Cron != "*/5 * * * *" || resp.IsEnabled || resp.NextRunAt != nil {
		t.Fatalf("Update = %+v, %v", resp, err)
	}
}

func TestJobService_TriggerRecordsResultAndPreventsOverlap(t *testing.T) {
	repo := newMemJobRepo()
	_, rdb := newJobTestRedis(t)
	release := make(chan struct{})
	svc := newJobTestService(t, repo, rdb, JobCatalog{
		{Key: "slow", DefaultCron: "0 * * * *", Timeout: time.Minute, Run: func(context.Context) (int64, error) {
			<-release
			return 7, nil
		}},
		{Key: "broken", DefaultCron: "0 * * * *", Timeout: time.Minute, Run: func(context.Context) (int64, error) {
			return 0, errors.New("storage unavailable")
		}},
		{Key: "panics", DefaultCron: "0 * * * *", Timeout: time.Minute, Run: func(context.Context) (int64, error) {
			panic("boom")
		}},
	})
	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}
	operator := JobOperator{ID: 1, Username: "system"}

	if _, err := svc.Trigger(ctx, "missing", operator); !errors.Is(err, apperror.ErrJobNotFound) {
		t.Fatalf("Trigger missing err = %v", err)
	}

	run, err := svc.Trigger(ctx, "slow", operator)
	if err != nil || run.Status != job.StatusRunning || run.Trigger != job.TriggerManual || run.Operator != "system" {
		t.Fatalf("Trigger = %+v, %v", run, err)
	}
	// 同一任务在执行中（锁在 Redis，任一副本都拿不到）
	other := newJobTestService(t, repo, rdb, JobCatalog{{Key: "slow", DefaultCron: "0 * * * *", Timeout: time.Minute, Run: func(context.Context) (int64, error) { return 0, nil }}})
	if _, err := other.Trigger(ctx, "slow", operator); !errors.Is(err, apperror.ErrJobAlreadyRunning) {
		t.Fatalf("overlapping Trigger err = %v", err)
	}
	close(release)
	if done := repo.waitRun(t, run.ID); done.Status != job.StatusSucceeded || done.Affected != 7 || done.FinishedAt == nil {
		t.Fatalf("finished run = %+v", done)
	}
	// 锁已释放，可以再次执行
	again, err := svc.Trigger(ctx, "slow", operator)
	if err != nil {
		t.Fatalf("Trigger after finish err = %v", err)
	}
	repo.waitRun(t, again.ID)

	failed, _ := svc.Trigger(ctx, "broken", operator)
	if done := repo.waitRun(t, failed.ID); done.Status != job.StatusFailed || done.Message != "storage unavailable" {
		t.Fatalf("failed run = %+v", done)
	}
	panicked, _ := svc.Trigger(ctx, "panics", operator)
	if done := repo.waitRun(t, panicked.ID); done.Status != job.StatusFailed || done.Message != "panic: boom" {
		t.Fatalf("panicked run = %+v", done)
	}
}

func TestJobService_TickRunsDueJobsOnceAcrossReplicas(t *testing.T) {
	repo := newMemJobRepo()
	_, rdb := newJobTestRedis(t)
	var calls atomic.Int32
	count := func(context.Context) (int64, error) { calls.Add(1); return 1, nil }
	catalog := JobCatalog{
		{Key: "hourly", DefaultCron: "0 * * * *", DefaultEnabled: true, Timeout: time.Minute, Run: count},
		{Key: "off", DefaultCron: "0 * * * *", DefaultEnabled: false, Timeout: time.Minute, Run: count},
		{Key: "daily", DefaultCron: "30 3 * * *", DefaultEnabled: true, Timeout: time.Minute, Run: count},
	}
	a := newJobTestService(t, repo, rdb, catalog)
	b := newJobTestService(t, repo, rdb, catalog)
	ctx := context.Background()
	if err := a.Start(ctx); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local)
	a.tick(ctx, at)
	b.tick(ctx, at) // 另一副本在同一调度点醒来
	a.Stop()
	b.Stop()
	if got := calls.Load(); got != 1 {
		t.Fatalf("due job ran %d times; want exactly once across replicas", got)
	}
	runs, _, _ := repo.ListRuns(ctx, 1, 10, "")
	if len(runs) != 1 || runs[0].JobKey != "hourly" || runs[0].Trigger != job.TriggerSchedule {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestJobService_DueMatchesWholeMinutes(t *testing.T) {
	svc := newJobTestService(t, newMemJobRepo(), nil, nil)
	at := time.Date(2026, 9, 23, 3, 30, 0, 0, time.Local)
	if !svc.due("30 3 * * *", at) || svc.due("31 3 * * *", at) || svc.due("not a cron", at) {
		t.Fatal("due() does not match the minute exactly")
	}
}

func TestJobService_ReconcileFailsOnlyUnlockedStaleRuns(t *testing.T) {
	repo := newMemJobRepo()
	_, rdb := newJobTestRedis(t)
	noop := func(context.Context) (int64, error) { return 0, nil }
	svc := newJobTestService(t, repo, rdb, JobCatalog{
		{Key: "crashed", DefaultCron: "0 * * * *", Timeout: time.Minute, Run: noop},
		{Key: "elsewhere", DefaultCron: "0 * * * *", Timeout: time.Minute, Run: noop},
	})
	ctx := context.Background()
	old := time.Now().Add(-time.Hour)
	crashed := &job.Run{JobKey: "crashed", Status: job.StatusRunning, StartedAt: old}
	elsewhere := &job.Run{JobKey: "elsewhere", Status: job.StatusRunning, StartedAt: old}
	_ = repo.CreateRun(ctx, crashed)
	_ = repo.CreateRun(ctx, elsewhere)
	// 另一个副本仍持有 elsewhere 的执行锁
	rdb.Set(ctx, svc.runLockKey("elsewhere"), "other-replica", time.Minute)

	if err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if run := repo.run(crashed.ID); run.Status != job.StatusFailed || run.Message != jobInterruptedMessage {
		t.Fatalf("crashed run = %+v", run)
	}
	if run := repo.run(elsewhere.ID); run.Status != job.StatusRunning {
		t.Fatalf("run held by another replica was touched: %+v", run)
	}
}

func TestJobService_StopCancelsRunningJobs(t *testing.T) {
	repo := newMemJobRepo()
	_, rdb := newJobTestRedis(t)
	started := make(chan struct{})
	svc := newJobTestService(t, repo, rdb, JobCatalog{
		{Key: "long", DefaultCron: "0 * * * *", Timeout: time.Hour, Run: func(ctx context.Context) (int64, error) {
			close(started)
			<-ctx.Done()
			return 3, ctx.Err()
		}},
	})
	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}
	run, err := svc.Trigger(ctx, "long", JobOperator{})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	svc.Stop()
	if done := repo.run(run.ID); done.Status != job.StatusFailed || done.Affected != 3 {
		t.Fatalf("cancelled run = %+v", done)
	}
	if n, _ := rdb.Exists(ctx, svc.runLockKey("long")).Result(); n != 0 {
		t.Fatal("run lock was not released on stop")
	}
	if _, err := svc.Trigger(ctx, "long", JobOperator{}); err == nil {
		t.Fatal("Trigger after Stop must be refused")
	}
	if repo.runCount() != 1 {
		t.Fatalf("runs = %d; want 1", repo.runCount())
	}
}

func TestZoneLabelNamesTheZone(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	moment := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"Asia/Shanghai": zoneLabel(moment.In(shanghai)),
		"UTC":           zoneLabel(moment),
		"UTC+08:00":     zoneLabel(moment.In(time.FixedZone("", 8*3600))),
		"UTC-03:30":     zoneLabel(moment.In(time.FixedZone("", -(3*3600 + 1800)))),
	}
	for want, got := range cases {
		if got != want {
			t.Errorf("zoneLabel = %q, want %q", got, want)
		}
	}
}
