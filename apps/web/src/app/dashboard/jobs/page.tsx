import PageContainer from '@/components/layout/page-container';
import JobsListingPage from '@/features/jobs/components/jobs-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.jobs')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function JobsPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const [canView, canViewRuns] = await Promise.all([
    serverHasPermission('/api/v1/admin/jobs:GET'),
    serverHasPermission('/api/v1/admin/job-runs:GET')
  ]);
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.jobs')}
      pageDescription={t('dashboard.jobsDescription')}
      access={canView}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <JobsListingPage canViewRuns={canViewRuns} />
    </PageContainer>
  );
}
