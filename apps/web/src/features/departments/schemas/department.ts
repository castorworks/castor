import { z } from 'zod';

export interface DepartmentFormMessages {
  codeInvalid: string;
  nameRequired: string;
}

/** Mirrors the backend rules: lowercase code, 1–100 character name. */
export function departmentSchema(messages: DepartmentFormMessages) {
  return z.object({
    parentId: z.string(),
    code: z
      .string()
      .trim()
      .regex(/^[a-z0-9][a-z0-9_-]*$/, messages.codeInvalid)
      .max(64, messages.codeInvalid),
    name: z.string().trim().min(1, messages.nameRequired).max(100, messages.nameRequired),
    sortOrder: z.number().int(),
    isEnabled: z.boolean()
  });
}

export type DepartmentFormValues = z.infer<ReturnType<typeof departmentSchema>>;
