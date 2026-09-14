import { z } from 'zod';
export function menuSchema(t: (key: string) => string) {
  return z
    .object({
      code: z
        .string()
        .regex(/^[a-z][a-z0-9_-]*$/, t('invalid'))
        .max(100, t('invalid')),
      kind: z.enum(['directory', 'page', 'action']),
      parentId: z.string(),
      en: z.string().trim().min(1, t('required')).max(100, t('invalid')),
      zh: z.string().trim().min(1, t('required')).max(100, t('invalid')),
      ja: z.string().trim().min(1, t('required')).max(100, t('invalid')),
      ko: z.string().trim().min(1, t('required')).max(100, t('invalid')),
      path: z.string(),
      icon: z.string(),
      sortOrder: z.number().int(),
      isEnabled: z.boolean(),
      accessMode: z.enum(['authenticated', 'permission']),
      permissions: z.array(z.object({ resourceId: z.number(), action: z.string() }))
    })
    .superRefine((value, ctx) => {
      if (value.kind === 'page' && !/^\/dashboard(?:\/[a-zA-Z0-9_-]+)+$/.test(value.path))
        ctx.addIssue({ code: 'custom', path: ['path'], message: t('pathHelp') });
      if (
        value.kind !== 'directory' &&
        (value.kind === 'action' || value.accessMode === 'permission') &&
        !value.permissions.length
      )
        ctx.addIssue({ code: 'custom', path: ['permissions'], message: t('permissionsRequired') });
    });
}
