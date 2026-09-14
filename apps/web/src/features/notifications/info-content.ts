import type { InfobarContent } from '@/components/ui/infobar';

export const adminNotificationsInfoContent: InfobarContent = {
  title: 'Notification Management',
  sections: [
    {
      title: 'Delivery Model',
      description:
        'Notifications are definitions; user_notifications records are the explicit inbox deliveries that track read and dismissed state.',
      links: []
    },
    {
      title: 'Recipient Control',
      description:
        'Global notices may be created without recipients, while targeted notices require explicit user IDs. Editing a targeted notice only replaces recipients when new IDs are provided.',
      links: []
    },
    {
      title: 'Operational Closure',
      description:
        'Create, update, delete, audit, and recipient inspection now share the same admin workflow so delivery state is visible after publishing.',
      links: []
    }
  ]
};
