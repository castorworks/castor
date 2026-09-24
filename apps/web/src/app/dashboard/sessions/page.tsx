import PageContainer from '@/components/layout/page-container';
import SessionsListingPage from '@/features/sessions/components/sessions-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.sessions')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function SessionsPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canView = await serverHasPermission('/api/v1/admin/sessions:GET');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.sessions')}
      pageDescription={t('dashboard.sessionsDescription')}
      access={canView}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <SessionsListingPage />
    </PageContainer>
  );
}
