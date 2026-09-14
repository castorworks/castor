import { z } from 'zod';

export const updateNameSchema = z.object({
  name: z.string().min(1, 'Name is required').max(50, 'Name must be 50 characters or less')
});

export const updatePasswordSchema = z
  .object({
    currentPassword: z.string().min(1, 'Current password is required'),
    newPassword: z.string().min(8, 'New password must be at least 8 characters'),
    confirmPassword: z.string().min(1, 'Please confirm your password')
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword']
  });

export type UpdateNameFormValues = z.infer<typeof updateNameSchema>;
export type UpdatePasswordFormValues = z.infer<typeof updatePasswordSchema>;
