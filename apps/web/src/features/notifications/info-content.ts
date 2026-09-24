import type { InfobarContent } from '@/components/ui/infobar';

export function getAdminNotificationsInfoContent(t: (key: string) => string): InfobarContent {
  return {
    title: t('notifications.admin.info.title'),
    sections: ['delivery', 'email', 'tracking'].map((section) => ({
      title: t(`notifications.admin.info.${section}Title`),
      description: t(`notifications.admin.info.${section}`),
      links: []
    }))
  };
}
