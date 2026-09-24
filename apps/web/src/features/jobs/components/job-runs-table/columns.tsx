'use client';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { Column, ColumnDef } from '@tanstack/react-table';
import { useLocale, useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { DictBadge } from '@/components/dict-badge';
import { useDict } from '@/hooks/use-dict';
import { formatDateTime } from '@/lib/format';
import { jobsQueryOptions } from '../../api/queries';
import type { JobRun } from '../../api/types';
import { useJobText } from '../../lib/job-name';

export const columnIds = [
  'job',
  'trigger',
  'runStatus',
  'startedAt',
  'duration',
  'affected',
  'message'
];

function durationSeconds(run: JobRun): number | null {
  if (!run.finishedAt) return null;
  return Math.max(0, (Date.parse(run.finishedAt) - Date.parse(run.startedAt)) / 1000);
}

export function useJobRunColumns(): ColumnDef<JobRun>[] {
  const t = useTranslations('jobs');
  const locale = useLocale();
  const dict = useDict();
  const text = useJobText();
  const { data: jobs } = useQuery(jobsQueryOptions());

  return [
    {
      id: 'job',
      accessorFn: (row) => row.jobKey,
      header: t('runs.columns.job'),
      cell: ({ row }) => <span className='font-medium'>{text.name(row.original.jobKey)}</span>,
      enableColumnFilter: true,
      enableSorting: false,
      meta: {
        label: t('runs.columns.job'),
        variant: 'select' as const,
        options: (jobs?.jobs ?? []).map((job) => ({ value: job.key, label: text.name(job.key) }))
      }
    },
    {
      id: 'trigger',
      accessorFn: (row) => row.trigger,
      header: t('runs.columns.trigger'),
      cell: ({ row }) => (
        <div className='flex flex-col gap-1'>
          <DictBadge type='job_trigger' value={row.original.trigger} />
          {row.original.operator && (
            <span className='text-muted-foreground text-xs'>@{row.original.operator}</span>
          )}
        </div>
      ),
      enableColumnFilter: true,
      enableSorting: false,
      meta: {
        label: t('runs.columns.trigger'),
        variant: 'multiSelect' as const,
        get options() {
          return dict.options('job_trigger');
        }
      }
    },
    {
      id: 'runStatus',
      accessorFn: (row) => row.status,
      header: t('runs.columns.status'),
      cell: ({ row }) => <DictBadge type='job_run_status' value={row.original.status} />,
      enableColumnFilter: true,
      enableSorting: false,
      meta: {
        label: t('runs.columns.status'),
        variant: 'multiSelect' as const,
        get options() {
          return dict.options('job_run_status');
        }
      }
    },
    {
      accessorKey: 'startedAt',
      header: ({ column }: { column: Column<JobRun, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('runs.columns.startedAt')} />
      ),
      cell: ({ row }) => (
        <span className='text-sm' suppressHydrationWarning>
          {formatDateTime(row.original.startedAt, { locale })}
        </span>
      )
    },
    {
      id: 'duration',
      header: t('runs.columns.duration'),
      cell: ({ row }) => {
        const seconds = durationSeconds(row.original);
        return (
          <span className='text-sm tabular-nums'>
            {seconds === null ? '—' : t('runs.seconds', { seconds: seconds.toFixed(1) })}
          </span>
        );
      },
      enableSorting: false
    },
    {
      accessorKey: 'affected',
      header: ({ column }: { column: Column<JobRun, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('runs.columns.affected')} />
      ),
      cell: ({ row }) => <span className='text-sm tabular-nums'>{row.original.affected}</span>
    },
    {
      id: 'message',
      header: t('runs.columns.message'),
      cell: ({ row }) => (
        <span
          className='text-muted-foreground block max-w-md truncate font-mono text-xs'
          title={row.original.message}
        >
          {row.original.message || '—'}
        </span>
      ),
      enableSorting: false
    }
  ];
}
