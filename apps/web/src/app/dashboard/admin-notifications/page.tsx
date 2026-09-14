import PageContainer from '@/components/layout/page-container';
import AdminNotificationListingPage from '@/features/notifications/components/admin-notification-listing';
import { NotificationFormSheetTrigger } from '@/features/notifications/components/notification-form-sheet';
import { adminNotificationsInfoContent } from '@/features/notifications/info-content';
import { searchParamsCache } from '@/lib/searchparams';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';
import type { SearchParams } from 'nuqs/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.notificationManagement')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function AdminNotificationsPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewNotifications = await serverHasPermission('/api/v1/admin/notifications:GET');
  const canManageNotifications = await serverHasPermission('/api/v1/admin/notifications:POST');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.notificationManagement')}
      pageDescription={t('notifications.admin.description')}
      infoContent={adminNotificationsInfoContent}
      access={canViewNotifications}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canManageNotifications ? <NotificationFormSheetTrigger /> : undefined}
    >
      <AdminNotificationListingPage />
    </PageContainer>
  );
}
