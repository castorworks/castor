import * as z from 'zod';

const DEFAULT_MIN_LENGTH = 6;

/** Factory for creating a user creation schema with dynamic password min length */
export function createUserSchema(minLength: number = DEFAULT_MIN_LENGTH) {
  return z.object({
    username: z.string().min(2, 'Username must be at least 2 characters'),
    name: z.string().optional(),
    password: z.string().min(minLength, `Password must be at least ${minLength} characters`)
  });
}

/** Factory for creating a user update schema with dynamic password min length */
export function updateUserSchema(minLength: number = DEFAULT_MIN_LENGTH) {
  return z.object({
    name: z.string().optional(),
    avatar: z.string().optional(),
    password: z
      .string()
      .min(minLength, `Password must be at least ${minLength} characters`)
      .optional(),
    enable: z.boolean().optional(),
    locked: z.boolean().optional(),
    accountExpireDate: z.string().nullable().optional(),
    credentialExpireDate: z.string().nullable().optional()
  });
}

export type CreateUserFormValues = z.infer<ReturnType<typeof createUserSchema>>;
export type UpdateUserFormValues = z.infer<ReturnType<typeof updateUserSchema>>;
