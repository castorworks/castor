import PageContainer from '@/components/layout/page-container';
import NotificationsListingPage from '@/features/notifications/components/notifications-listing';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.notifications')
  };
}

export default async function NotificationsPage() {
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('notifications.title')}
      pageDescription={t('notifications.description')}
    >
      <NotificationsListingPage />
    </PageContainer>
  );
}
