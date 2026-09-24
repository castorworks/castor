import PageContainer from '@/components/layout/page-container';
import AuditLogsListingPage from '@/features/audit-logs/components/audit-logs-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.auditLogs')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function AuditLogsPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewAuditLogs = await serverHasPermission('/api/v1/admin/audit-logs:GET');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.auditLogs')}
      pageDescription={t('dashboard.auditLogsDescription')}
      access={canViewAuditLogs}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <AuditLogsListingPage />
    </PageContainer>
  );
}
