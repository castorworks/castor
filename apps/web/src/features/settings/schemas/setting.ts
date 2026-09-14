import { z } from 'zod';

export const createSettingSchema = z.object({
  key: z.string().min(1, 'Key is required'),
  name: z.string().min(1, 'Name is required'),
  value: z.string().optional(),
  type: z.enum(['STRING', 'NUMBER', 'BOOL', 'JSON', 'ARRAY', 'SECRET']).optional(),
  category: z.string().optional(),
  description: z.string().optional(),
  defaultVal: z.string().optional(),
  isPublic: z.boolean().optional(),
  sortOrder: z.number().optional()
});

export const updateSettingSchema = z.object({
  name: z.string().optional(),
  value: z.string().optional(),
  description: z.string().optional(),
  isPublic: z.boolean().optional(),
  sortOrder: z.number().optional()
});

export type CreateSettingFormValues = z.infer<typeof createSettingSchema>;
export type UpdateSettingFormValues = z.infer<typeof updateSettingSchema>;
