import { z } from 'zod';

export const createAssetSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  tags: z.string().optional(),
  folderPath: z.string().optional(),
  isPublic: z.boolean().default(false)
});

export const updateAssetSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  tags: z.string().optional(),
  folderPath: z.string().optional(),
  isPublic: z.boolean().optional()
});

export type CreateAssetFormValues = z.infer<typeof createAssetSchema>;
export type UpdateAssetFormValues = z.infer<typeof updateAssetSchema>;
