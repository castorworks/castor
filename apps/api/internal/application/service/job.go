package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

// JobDefinition 是在代码里注册的一个定时任务。运营只能调整 cron 与启停、手动执行、查看记录；
// 任务做什么只由 Run 决定，数据库里永远不会出现可被反射调用的函数名。
type JobDefinition struct {
	// Key 任务标识，同时是前端翻译键 jobs.catalog.{key} 的一段，只用小驼峰字母
	Key         string
	DefaultCron string
	// DefaultEnabled 首次注册时是否启用
	DefaultEnabled bool
	// Timeout 单次执行上限，超时后 ctx 被取消
	Timeout time.Duration
	// Run 执行任务，返回处理的条数
	Run func(ctx context.Context) (int64, error)
}

// JobCatalog 本进程注册的全部定时任务
type JobCatalog []JobDefinition

// 任务标识
const (
	JobSweepOrphanAttachments    = "sweepOrphanAttachments"
	JobPurgeExpiredSessions      = "purgeExpiredSessions"
	JobDeliverNotificationEmails = "deliverNotificationEmails"
)

// sessionRetention 已撤销或已过期的授权会话保留多久后删除；期间仍可在排查时查到它们。
const sessionRetention = 30 * 24 * time.Hour

// NewJobCatalog 注册内置定时任务。新增任务：在这里追加定义，并在前端 jobs.catalog 补四语名称与说明。
func NewJobCatalog(assets AssetService, sweep AssetSweepPolicy, sessions permission.AuthorizationRepository, emails NotificationEmailService) JobCatalog {
	var catalog JobCatalog
	// 配置里关闭了孤儿附件清扫（宽限期为 0）时不注册该任务，界面上也就看不到它。
	if sweep.OrphanTTL > 0 {
		catalog = append(catalog, JobDefinition{
			Key:            JobSweepOrphanAttachments,
			DefaultCron:    "0 * * * *",
			DefaultEnabled: true,
			Timeout:        30 * time.Minute,
			Run: func(ctx context.Context) (int64, error) {
				n, err := assets.SweepOrphanAttachments(ctx, sweep.OrphanTTL)
				return int64(n), err
			},
		})
	}
	catalog = append(catalog, JobDefinition{
		Key:            JobPurgeExpiredSessions,
		DefaultCron:    "30 3 * * *",
		DefaultEnabled: true,
		Timeout:        10 * time.Minute,
		Run: func(ctx context.Context) (int64, error) {
			return sessions.PurgeSessions(ctx, time.Now().Add(-sessionRetention))
		},
	})
	// 没有配置邮件服务时通知不会登记邮件，也就不需要这个任务。
	if emails != nil && emails.Available() {
		catalog = append(catalog, JobDefinition{
			Key:            JobDeliverNotificationEmails,
			DefaultCron:    "* * * * *",
			DefaultEnabled: true,
			Timeout:        5 * time.Minute,
			Run:            emails.DeliverDue,
		})
	}
	return catalog
}

// JobOperator 手动执行任务的操作者
type JobOperator struct {
	ID       uint
	Username string
}

// JobService 定时任务：运营设置、手动执行、执行记录与调度循环。
type JobService interface {
	List(ctx context.Context) (*dto.JobListResp, error)
	Update(ctx context.Context, key string, req dto.JobUpdateRequest) (*dto.JobResp, error)
	// Trigger 立即在后台执行一次，返回刚创建的执行记录；任务正在执行（任一副本）时返回 ErrJobAlreadyRunning。
	Trigger(ctx context.Context, key string, operator JobOperator) (*job.Run, error)
	ListRuns(ctx context.Context, page, size int, order string, opts ...query.Option) ([]job.Run, int64, error)

	// Start 补齐任务设置并启动调度循环；Stop 停止调度、取消进行中的执行并等它们收尾。
	// 必须在关闭数据库与 Redis 之前调用 Stop。
	Start(ctx context.Context) error
	Stop()
}

const (
	// jobRunsKept 每个任务保留的执行记录条数
	jobRunsKept = 100
	// jobLockMargin 执行锁在任务超时之外多留的时间，覆盖收尾写库
	jobLockMargin = time.Minute
	// jobTickLockTTL 调度点锁的存活时间：只需长于各副本时钟偏差
	jobTickLockTTL = 10 * time.Minute
	// jobMessageMaxLen 失败原因的最大长度（与 job_runs.message 列宽一致）
	jobMessageMaxLen = 500
	// jobInterruptedMessage 执行它的进程已退出时写入的失败原因
	jobInterruptedMessage = "interrupted: the process running this job exited before it finished"
)

// cronParser 只接受标准 5 段表达式（分 时 日 月 周），不接受 @every 一类描述符：
// 调度循环按整分钟触发，秒级或相对间隔的表达式无法兑现。
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// releaseLockScript 只删除自己持有的锁，避免锁过期后误删别的副本刚拿到的锁。
var releaseLockScript = redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`)

// ParseCron 校验并规范化 cron 表达式（多余空白压成单个空格）。
func ParseCron(spec string) (string, cron.Schedule, error) {
	fields := strings.Fields(spec)
	// 显式要求 5 段：拒绝 TZ= 前缀（时区统一取进程时区）与描述符。
	if len(fields) != 5 || strings.ContainsAny(spec, "=@") {
		return "", nil, apperror.ErrInvalidCron
	}
	normalized := strings.Join(fields, " ")
	schedule, err := cronParser.Parse(normalized)
	if err != nil {
		return "", nil, apperror.ErrInvalidCron
	}
	// 永远不会触发的表达式（如 2 月 30 日）视为无效。
	if schedule.Next(time.Now()).IsZero() {
		return "", nil, apperror.ErrInvalidCron
	}
	return normalized, schedule, nil
}

type jobService struct {
	repo    job.Repository
	catalog map[string]JobDefinition
	order   []string
	redis   redis.UniversalClient
	keys    rediskey.Namespace
	now     func() time.Time

	mu      sync.Mutex
	stopped bool
	baseCtx context.Context
	cancel  context.CancelFunc
	running sync.WaitGroup
}

// NewJobService 创建定时任务服务
func NewJobService(repo job.Repository, catalog JobCatalog, rdb redis.UniversalClient, ns rediskey.Namespace) JobService {
	svc := &jobService{
		repo:    repo,
		catalog: make(map[string]JobDefinition, len(catalog)),
		redis:   rdb,
		keys:    ns,
		now:     time.Now,
	}
	for _, def := range catalog {
		svc.catalog[def.Key] = def
		svc.order = append(svc.order, def.Key)
	}
	svc.baseCtx, svc.cancel = context.WithCancel(context.Background())
	return svc
}

func (svc *jobService) List(ctx context.Context) (*dto.JobListResp, error) {
	items, err := svc.repo.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := svc.repo.LatestRuns(ctx)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]job.Job, len(items))
	for _, item := range items {
		byKey[item.Key] = item
	}
	resp := &dto.JobListResp{TimeZone: zoneLabel(svc.now()), Jobs: make([]dto.JobResp, 0, len(svc.order))}
	// 只列出本进程注册的任务：代码里下线的任务留在库里的设置不再展示。
	for _, key := range svc.order {
		item, ok := byKey[key]
		if !ok {
			continue
		}
		var last *job.Run
		if run, ok := latest[key]; ok {
			last = &run
		}
		resp.Jobs = append(resp.Jobs, svc.toResp(item, last))
	}
	return resp, nil
}

func (svc *jobService) toResp(item job.Job, last *job.Run) dto.JobResp {
	def := svc.catalog[item.Key]
	resp := dto.JobResp{Job: item, DefaultCron: def.DefaultCron, TimeoutSeconds: int64(def.Timeout / time.Second), LastRun: last}
	if item.IsEnabled {
		if _, schedule, err := ParseCron(item.Cron); err == nil {
			next := schedule.Next(svc.now())
			resp.NextRunAt = &next
		}
	}
	return resp
}

func (svc *jobService) lookup(ctx context.Context, key string) (JobDefinition, *job.Job, error) {
	def, ok := svc.catalog[key]
	if !ok {
		return JobDefinition{}, nil, apperror.ErrJobNotFound
	}
	item, err := svc.repo.GetJob(ctx, key)
	if errors.Is(err, shared.ErrNotFound) {
		return JobDefinition{}, nil, apperror.ErrJobNotFound
	}
	if err != nil {
		return JobDefinition{}, nil, err
	}
	return def, item, nil
}

func (svc *jobService) Update(ctx context.Context, key string, req dto.JobUpdateRequest) (*dto.JobResp, error) {
	_, item, err := svc.lookup(ctx, key)
	if err != nil {
		return nil, err
	}
	spec, _, err := ParseCron(req.Cron)
	if err != nil {
		return nil, err
	}
	item.Cron, item.IsEnabled = spec, req.IsEnabled
	if err := svc.repo.UpdateJob(ctx, item); err != nil {
		return nil, err
	}
	latest, err := svc.repo.LatestRuns(ctx)
	if err != nil {
		return nil, err
	}
	var last *job.Run
	if run, ok := latest[key]; ok {
		last = &run
	}
	resp := svc.toResp(*item, last)
	return &resp, nil
}

func (svc *jobService) Trigger(ctx context.Context, key string, operator JobOperator) (*job.Run, error) {
	def, _, err := svc.lookup(ctx, key)
	if err != nil {
		return nil, err
	}
	// 手动执行不看启停：停用只影响按计划调度。
	return svc.launch(ctx, def, job.TriggerManual, operator)
}

func (svc *jobService) ListRuns(ctx context.Context, page, size int, order string, opts ...query.Option) ([]job.Run, int64, error) {
	return svc.repo.ListRuns(ctx, page, size, order, opts...)
}

func (svc *jobService) runLockKey(key string) string {
	return svc.keys.Key("job:run:" + key)
}

func (svc *jobService) tickLockKey(key string, at time.Time) string {
	return svc.keys.Key(fmt.Sprintf("job:tick:%s:%d", key, at.Unix()))
}

func newLockToken() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// launch 取得执行锁、写入 RUNNING 记录并在后台执行。锁跨副本保证同一任务不重叠执行。
func (svc *jobService) launch(ctx context.Context, def JobDefinition, trigger job.Trigger, operator JobOperator) (*job.Run, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.stopped {
		return nil, apperror.ErrJobAlreadyRunning
	}

	token := newLockToken()
	lockKey := svc.runLockKey(def.Key)
	acquired, err := svc.redis.SetNX(ctx, lockKey, token, def.Timeout+jobLockMargin).Result()
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, apperror.ErrJobAlreadyRunning
	}

	run := &job.Run{
		JobKey: def.Key, Trigger: trigger, Status: job.StatusRunning, StartedAt: svc.now(),
		Operator: operator.Username, OperatorID: operator.ID,
	}
	if err := svc.repo.CreateRun(ctx, run); err != nil {
		svc.releaseLock(lockKey, token)
		return nil, err
	}

	svc.running.Add(1)
	go func() {
		defer svc.running.Done()
		svc.execute(def, run.ID, lockKey, token)
	}()
	created := *run
	return &created, nil
}

func (svc *jobService) execute(def JobDefinition, runID uint, lockKey, token string) {
	defer svc.releaseLock(lockKey, token)

	ctx, cancel := context.WithTimeout(svc.baseCtx, def.Timeout)
	affected, err := safeRun(ctx, def)
	cancel()

	status, message := job.StatusSucceeded, ""
	if err != nil {
		status, message = job.StatusFailed, truncateJobMessage(err.Error())
		log.WarnCtx(context.Background()).Err(err).Str("job", def.Key).Msg("Scheduled job failed")
	}
	// 收尾写库不用任务的 ctx：任务因超时或停机被取消时，记录仍要落库。
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finishCancel()
	if err := svc.repo.FinishRun(finishCtx, runID, status, affected, message, svc.now()); err != nil {
		log.ErrCtx(context.Background(), err).Str("job", def.Key).Msg("Failed to record scheduled job result")
	}
	if err := svc.repo.PruneRuns(finishCtx, def.Key, jobRunsKept); err != nil {
		log.WarnCtx(context.Background()).Err(err).Str("job", def.Key).Msg("Failed to prune scheduled job runs")
	}
}

// safeRun 把任务里的 panic 转成失败，不让一个任务拖垮进程。
func safeRun(ctx context.Context, def JobDefinition) (affected int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return def.Run(ctx)
}

func truncateJobMessage(message string) string {
	runes := []rune(message)
	if len(runes) <= jobMessageMaxLen {
		return message
	}
	return string(runes[:jobMessageMaxLen-1]) + "…"
}

func (svc *jobService) releaseLock(lockKey, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := releaseLockScript.Run(ctx, svc.redis, []string{lockKey}, token).Err(); err != nil {
		log.WarnCtx(context.Background()).Err(err).Str("lock", lockKey).Msg("Failed to release scheduled job lock")
	}
}

func (svc *jobService) Start(ctx context.Context) error {
	defaults := make([]job.Job, 0, len(svc.order))
	for _, key := range svc.order {
		def := svc.catalog[key]
		if _, _, err := ParseCron(def.DefaultCron); err != nil {
			return fmt.Errorf("job %s: invalid default cron %q", key, def.DefaultCron)
		}
		defaults = append(defaults, job.Job{Key: key, Cron: def.DefaultCron, IsEnabled: def.DefaultEnabled})
	}
	if err := svc.repo.EnsureJobs(ctx, defaults); err != nil {
		return err
	}
	svc.reconcile(ctx)

	svc.running.Add(1)
	go func() {
		defer svc.running.Done()
		svc.loop()
	}()
	return nil
}

func (svc *jobService) Stop() {
	svc.mu.Lock()
	svc.stopped = true
	svc.mu.Unlock()
	svc.cancel()
	svc.running.Wait()
}

// loop 在每个整分钟醒来一次，执行当分钟到点的任务。进程暂停或忙碌错过的调度点直接跳过，不补跑。
func (svc *jobService) loop() {
	for {
		now := svc.now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		timer := time.NewTimer(next.Sub(now))
		select {
		case <-svc.baseCtx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		svc.tick(svc.baseCtx, next)
	}
}

// tick 执行 at（整分钟）到点的已启用任务，并清理没有进程在执行的 RUNNING 记录。
// 每个调度点先抢调度点锁，确保多副本下同一调度点只执行一次。
func (svc *jobService) tick(ctx context.Context, at time.Time) {
	svc.reconcile(ctx)
	items, err := svc.repo.ListJobs(ctx)
	if err != nil {
		log.WarnCtx(context.Background()).Err(err).Msg("Failed to load scheduled jobs")
		return
	}
	for _, item := range items {
		def, ok := svc.catalog[item.Key]
		if !ok || !item.IsEnabled || !svc.due(item.Cron, at) {
			continue
		}
		claimed, err := svc.redis.SetNX(ctx, svc.tickLockKey(item.Key, at), "1", jobTickLockTTL).Result()
		if err != nil {
			log.WarnCtx(context.Background()).Err(err).Str("job", item.Key).Msg("Failed to claim scheduled job tick")
			continue
		}
		if !claimed {
			continue
		}
		if _, err := svc.launch(ctx, def, job.TriggerSchedule, JobOperator{}); err != nil {
			// 上一次执行还没结束：跳过本次，不排队。
			if !errors.Is(err, apperror.ErrJobAlreadyRunning) {
				log.WarnCtx(context.Background()).Err(err).Str("job", item.Key).Msg("Failed to start scheduled job")
			}
		}
	}
}

func (svc *jobService) due(spec string, at time.Time) bool {
	_, schedule, err := ParseCron(spec)
	if err != nil {
		return false
	}
	return schedule.Next(at.Add(-time.Second)).Equal(at)
}

// reconcile 把没有任何进程持有执行锁、却仍是 RUNNING 的记录标为中断失败（进程崩溃或被强杀）。
// 只处理检查之前开始的记录：检查之后别的副本新拿锁写下的记录不受影响。
func (svc *jobService) reconcile(ctx context.Context) {
	for _, key := range svc.order {
		checkedAt := svc.now()
		held, err := svc.redis.Exists(ctx, svc.runLockKey(key)).Result()
		if err != nil {
			log.WarnCtx(context.Background()).Err(err).Str("job", key).Msg("Failed to check scheduled job lock")
			continue
		}
		if held > 0 {
			continue
		}
		// 退一分钟容忍副本间的时钟偏差：宁可晚一轮收尾，也不误伤别的副本刚开始的执行。
		if _, err := svc.repo.FailStaleRuns(ctx, key, jobInterruptedMessage, checkedAt.Add(-time.Minute)); err != nil {
			log.WarnCtx(context.Background()).Err(err).Str("job", key).Msg("Failed to close interrupted scheduled job runs")
		}
	}
}

// zoneLabel 给界面标注 cron 所用的时区。进程没设 TZ 时 Go 把本地时区叫 "Local"，
// 对运维没有意义，此时改用 UTC 偏移（如 UTC+08:00）。
func zoneLabel(t time.Time) string {
	if name := t.Location().String(); name != "Local" && name != "" {
		return name
	}
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign, offset = "-", -offset
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, offset/3600, offset%3600/60)
}
