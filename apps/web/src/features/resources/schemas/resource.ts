import * as z from 'zod';

export const resourceValidationMessages = {
  codeMin: 'resources.validation.codeMin',
  nameMin: 'resources.validation.nameMin',
  pathRequired: 'resources.validation.pathRequired',
  actionsRequired: 'resources.validation.actionsRequired',
  categoryRequired: 'resources.validation.categoryRequired'
} as const;

type ResourceValidationMessages = Record<keyof typeof resourceValidationMessages, string>;

const defaultResourceValidationMessages: ResourceValidationMessages = resourceValidationMessages;

/** Schema for creating a new resource */
export function getCreateResourceSchema(
  messages: ResourceValidationMessages = defaultResourceValidationMessages
) {
  return z.object({
    code: z.string().min(2, messages.codeMin),
    name: z.string().min(2, messages.nameMin),
    description: z.string().optional(),
    path: z.string().min(1, messages.pathRequired),
    actions: z.array(z.string()).min(1, messages.actionsRequired),
    category: z.string().min(1, messages.categoryRequired),
    module: z.string().optional(),
    sortOrder: z.number().optional()
  });
}

export const createResourceSchema = getCreateResourceSchema();

/** Schema for updating an existing resource */
export function getUpdateResourceSchema(
  messages: ResourceValidationMessages = defaultResourceValidationMessages
) {
  return z.object({
    name: z.string().min(2, messages.nameMin),
    description: z.string().optional(),
    path: z.string().min(1, messages.pathRequired),
    actions: z.array(z.string()).min(1, messages.actionsRequired),
    category: z.string().min(1, messages.categoryRequired),
    module: z.string().optional(),
    sortOrder: z.number().optional(),
    isEnabled: z.boolean().optional()
  });
}

export const updateResourceSchema = getUpdateResourceSchema();

export type CreateResourceFormValues = z.infer<typeof createResourceSchema>;
export type UpdateResourceFormValues = z.infer<typeof updateResourceSchema>;
