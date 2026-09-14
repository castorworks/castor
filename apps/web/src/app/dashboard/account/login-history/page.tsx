import PageContainer from '@/components/layout/page-container';
import AccountLoginHistoryListingPage from '@/features/account/components/account-login-history-listing';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('loginHistory.title')
  };
}

export default async function Page() {
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('loginHistory.title')}
      pageDescription={t('dashboard.loginHistoriesDescription')}
    >
      <AccountLoginHistoryListingPage />
    </PageContainer>
  );
}
