import PageContainer from '@/components/layout/page-container';
import { SSOListing } from '@/features/sso/components/sso-listing';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return { title: t('nav.sso') };
}

export default async function SSOPage() {
  const canView = await serverHasPermission('/api/v1/admin/oidc-providers:GET');
  const t = await getTranslations();
  return (
    <PageContainer
      pageTitle={t('nav.sso')}
      pageDescription={t('dashboard.ssoDescription')}
      access={canView}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <SSOListing />
    </PageContainer>
  );
}
