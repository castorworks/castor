import type { InfobarContent } from '@/components/ui/infobar';

export function getUsersInfoContent(t: (key: string) => string): InfobarContent {
  return {
    title: t('users.info.title'),
    sections: ['overview', 'roles', 'sessions'].map((section) => ({
      title: t(`users.info.${section}Title`),
      description: t(`users.info.${section}`),
      links: []
    }))
  };
}
