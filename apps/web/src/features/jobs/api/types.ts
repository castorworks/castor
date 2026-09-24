// ============================================================
// Scheduled job types — aligned with castor backend dto.JobResp / job.Run
// ============================================================

/** A run's status; values of the `job_run_status` dictionary. */
export type JobRunStatus = 'RUNNING' | 'SUCCEEDED' | 'FAILED';

/** What started a run; values of the `job_trigger` dictionary. */
export type JobTrigger = 'SCHEDULE' | 'MANUAL';

export interface JobRun {
  id: number;
  jobKey: string;
  trigger: JobTrigger;
  status: JobRunStatus;
  startedAt: string;
  finishedAt: string | null;
  /** Number of records the job processed. */
  affected: number;
  /** Failure reason (technical detail, not localized); empty on success. */
  message: string;
  /** Who started a manual run; empty for scheduled runs. */
  operator: string;
  operatorId: number;
}

/**
 * A job registered in backend code. Operators can only change its cron
 * expression and whether it runs on schedule; what it does lives in code.
 */
export interface Job {
  key: string;
  cron: string;
  isEnabled: boolean;
  updatedAt: string;
  updatedBy: number;
  defaultCron: string;
  timeoutSeconds: number;
  /** Next scheduled run; null when disabled. */
  nextRunAt: string | null;
  lastRun: JobRun | null;
}

export interface JobList {
  /** Time zone the API evaluates cron expressions in. */
  timeZone: string;
  jobs: Job[];
}

export interface JobUpdatePayload {
  cron: string;
  isEnabled: boolean;
}

export interface JobRunFilters {
  page?: number;
  pageSize?: number;
  jobKey?: string;
  status?: string;
  trigger?: string;
  sort?: string;
}

export interface JobRunsPageResult {
  list: JobRun[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}
