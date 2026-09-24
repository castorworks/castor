import PageContainer from '@/components/layout/page-container';
import { ApiDocsViewer } from '@/features/api-docs/components/api-docs-viewer';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.apiDocs')
  };
}

export default async function ApiDocsPage() {
  const canView = await serverHasPermission('/api/v1/admin/openapi.json:GET');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.apiDocs')}
      pageDescription={t('dashboard.apiDocsDescription')}
      access={canView}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <ApiDocsViewer />
    </PageContainer>
  );
}
