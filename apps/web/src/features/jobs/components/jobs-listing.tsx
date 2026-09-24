import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getTranslations } from 'next-intl/server';
import { Heading } from '@/components/ui/heading';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { jobRunsQueryOptions, jobsQueryOptions } from '../api/queries';
import { JobList } from './job-list';
import { JobRunsTable } from './job-runs-table';

interface JobsListingPageProps {
  /** The run history needs its own permission (`/api/v1/admin/job-runs:GET`). */
  canViewRuns: boolean;
}

export default async function JobsListingPage({ canViewRuns }: JobsListingPageProps) {
  const t = await getTranslations('jobs');
  const page = searchParamsCache.get('page');
  const pageLimit = searchParamsCache.get('perPage');
  const job = searchParamsCache.get('job');
  const trigger = searchParamsCache.get('trigger');
  const runStatus = searchParamsCache.get('runStatus');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(job && { jobKey: job }),
    ...(trigger && { trigger }),
    ...(runStatus && { status: runStatus }),
    ...(sort && { sort })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await Promise.all([
      queryClient.prefetchQuery(jobsQueryOptions({ headers })),
      canViewRuns ? queryClient.prefetchQuery(jobRunsQueryOptions(filters, { headers })) : null
    ]);
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <div className='space-y-8'>
        <JobList />
        {/* DataTable 填满父元素的高度：放在卡片下方的正常文档流里时要给它一个高度，否则表格塌成 0。 */}
        {canViewRuns && (
          <section className='flex h-[40rem] flex-col gap-4'>
            <Heading title={t('runs.title')} description={t('runs.description')} />
            <JobRunsTable />
          </section>
        )}
      </div>
    </HydrationBoundary>
  );
}
