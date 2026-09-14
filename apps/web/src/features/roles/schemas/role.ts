import * as z from 'zod';

export const createRoleSchema = z.object({
  code: z
    .string()
    .min(2, 'Code must be at least 2 characters')
    .regex(/^[a-z0-9_-]+$/, 'Code must be lowercase letters, numbers, hyphens, or underscores'),
  name: z.string().min(2, 'Name must be at least 2 characters'),
  description: z.string().optional()
});

export type CreateRoleFormValues = z.infer<typeof createRoleSchema>;

export const updateRoleSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  description: z.string().optional()
});

export type UpdateRoleFormValues = z.infer<typeof updateRoleSchema>;
