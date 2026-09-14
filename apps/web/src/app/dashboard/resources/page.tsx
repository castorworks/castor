import PageContainer from '@/components/layout/page-container';
import ResourceListingPage from '@/features/resources/components/resource-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { ResourceFormSheetTrigger } from '@/features/resources/components/resource-form-sheet';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();

  return {
    title: t('nav.resources')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function ResourcesPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewResources = await serverHasPermission('/api/v1/admin/resources:GET');
  const canManageResources = await serverHasPermission('/api/v1/admin/resources:POST');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.resources')}
      pageDescription={t('dashboard.resourcesDescription')}
      access={canViewResources}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canManageResources ? <ResourceFormSheetTrigger /> : undefined}
    >
      <ResourceListingPage />
    </PageContainer>
  );
}
