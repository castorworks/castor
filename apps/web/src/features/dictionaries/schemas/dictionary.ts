import { z } from 'zod';

export const createDictTypeSchema = z.object({
  code: z
    .string()
    .min(1, 'Code is required')
    .regex(/^[a-z0-9_]+$/, 'Code must be lowercase letters, numbers, and underscores only'),
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional(),
  sortOrder: z.number().optional()
});

export const createDictItemSchema = z.object({
  label: z.string().min(1, 'Label is required'),
  value: z.string().min(1, 'Value is required'),
  description: z.string().optional(),
  color: z.string().optional(),
  icon: z.string().optional(),
  isDefault: z.boolean().optional(),
  sortOrder: z.number().optional()
});

export type CreateDictTypeFormValues = z.infer<typeof createDictTypeSchema>;
export type CreateDictItemFormValues = z.infer<typeof createDictItemSchema>;
