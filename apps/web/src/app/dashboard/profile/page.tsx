import PageContainer from '@/components/layout/page-container';
import ProfileListingPage from '@/features/account/components/profile-listing';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.profile')
  };
}

export default async function Page() {
  const t = await getTranslations();

  return (
    <PageContainer pageTitle={t('nav.profile')} pageDescription={t('dashboard.profileDescription')}>
      <ProfileListingPage />
    </PageContainer>
  );
}
