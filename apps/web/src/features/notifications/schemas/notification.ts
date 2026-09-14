import { z } from 'zod';

interface NotificationSchemaOptions {
  requireRecipients?: boolean;
}

const RECIPIENT_IDS_PATTERN = /^\s*\d+(\s*,\s*\d+)*\s*$/;

export function createNotificationSchema(
  t: (key: string) => string,
  options: NotificationSchemaOptions = {}
) {
  const requireRecipients = options.requireRecipients ?? true;

  return z
    .object({
      title: z.string().min(1, t('validation.titleRequired')).max(200, t('validation.titleMax')),
      content: z.string().optional(),
      type: z.enum(['SYSTEM', 'ANNOUNCE', 'MESSAGE', 'ALERT', 'TASK']),
      level: z.enum(['INFO', 'SUCCESS', 'WARNING', 'ERROR']),
      link: z.string().max(500, t('validation.linkMax')).optional(),
      extra: z.string().optional(),
      isGlobal: z.boolean(),
      userIds: z.string().optional(),
      expireAt: z.string().optional()
    })
    .superRefine((value, ctx) => {
      const recipientText = value.userIds?.trim();
      if (requireRecipients && !recipientText) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['userIds'],
          message: t('validation.recipientsRequired')
        });
        return;
      }

      if (recipientText && !RECIPIENT_IDS_PATTERN.test(recipientText)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['userIds'],
          message: t('validation.recipientsInvalid')
        });
      }
    });
}

export type CreateNotificationFormValues = z.infer<ReturnType<typeof createNotificationSchema>>;
