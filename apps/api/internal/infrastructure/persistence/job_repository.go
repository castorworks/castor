package persistence

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"github.com/castorworks/castor/internal/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type jobRepository struct{ db *gorm.DB }

func NewJobRepository(db *gorm.DB) job.Repository {
	return &jobRepository{db: db}
}

func (r *jobRepository) EnsureJobs(ctx context.Context, defaults []job.Job) error {
	if len(defaults) == 0 {
		return nil
	}
	rows := make([]models.ScheduledJobModel, len(defaults))
	for i, d := range defaults {
		rows[i] = models.ScheduledJobModel{Key: d.Key, Cron: d.Cron, IsEnabled: d.IsEnabled}
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *jobRepository) ListJobs(ctx context.Context) ([]job.Job, error) {
	var rows []models.ScheduledJobModel
	if err := r.db.WithContext(ctx).Order("key").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]job.Job, len(rows))
	for i := range rows {
		result[i] = rows[i].ToEntity()
	}
	return result, nil
}

func (r *jobRepository) GetJob(ctx context.Context, key string) (*job.Job, error) {
	var row models.ScheduledJobModel
	if err := translateError(r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error); err != nil {
		return nil, err
	}
	item := row.ToEntity()
	return &item, nil
}

func (r *jobRepository) UpdateJob(ctx context.Context, item *job.Job) error {
	row := models.ScheduledJobModel{Key: item.Key}
	if err := r.db.WithContext(ctx).Model(&row).Select("Cron", "IsEnabled", "UpdatedAt", "UpdatedBy").
		Updates(models.ScheduledJobModel{Cron: item.Cron, IsEnabled: item.IsEnabled, UpdatedAt: time.Now()}).Error; err != nil {
		return err
	}
	updated, err := r.GetJob(ctx, item.Key)
	if err != nil {
		return err
	}
	*item = *updated
	return nil
}

func (r *jobRepository) CreateRun(ctx context.Context, run *job.Run) error {
	row := models.JobRunModel{
		JobKey: run.JobKey, Trigger: string(run.Trigger), Status: string(run.Status), StartedAt: run.StartedAt,
		Message: run.Message, Operator: run.Operator, OperatorID: run.OperatorID,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	run.ID = row.ID
	return nil
}

func (r *jobRepository) FinishRun(ctx context.Context, id uint, status job.Status, affected int64, message string, finishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.JobRunModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(status), "affected": affected, "message": message, "finished_at": finishedAt}).Error
}

func (r *jobRepository) ListRuns(ctx context.Context, page, size int, order string, opts ...query.Option) ([]job.Run, int64, error) {
	rows, total, err := PaginatedQuery[models.JobRunModel](ctx, r.db, page, size, order, opts...)
	if err != nil {
		return nil, 0, err
	}
	result := make([]job.Run, len(rows))
	for i := range rows {
		result[i] = rows[i].ToEntity()
	}
	return result, total, nil
}

func (r *jobRepository) LatestRuns(ctx context.Context) (map[string]job.Run, error) {
	var rows []models.JobRunModel
	if err := r.db.WithContext(ctx).Raw(
		`SELECT DISTINCT ON (job_key) * FROM job_runs ORDER BY job_key, started_at DESC, id DESC`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]job.Run, len(rows))
	for i := range rows {
		result[rows[i].JobKey] = rows[i].ToEntity()
	}
	return result, nil
}

func (r *jobRepository) FailStaleRuns(ctx context.Context, key, message string, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.JobRunModel{}).
		Where("job_key = ? AND status = ? AND started_at < ?", key, string(job.StatusRunning), before).
		Updates(map[string]any{"status": string(job.StatusFailed), "message": message, "finished_at": before})
	return result.RowsAffected, result.Error
}

func (r *jobRepository) PruneRuns(ctx context.Context, key string, keep int) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM job_runs WHERE job_key = ? AND id NOT IN (
		SELECT id FROM job_runs WHERE job_key = ? ORDER BY started_at DESC, id DESC LIMIT ?)`, key, key, keep).Error
}
