import { z } from 'zod';

export interface JobFormMessages {
  cronInvalid: string;
}

/**
 * A light client check: five space-separated fields, no descriptors or TZ prefix.
 * The backend parses the expression and is the authority (ErrInvalidCron).
 */
export function jobSchema(messages: JobFormMessages) {
  return z.object({
    cron: z
      .string()
      .trim()
      .max(100, messages.cronInvalid)
      .refine((value) => {
        const fields = value.split(/\s+/).filter(Boolean);
        return fields.length === 5 && !/[=@]/.test(value);
      }, messages.cronInvalid),
    isEnabled: z.boolean()
  });
}

export type JobFormValues = z.infer<ReturnType<typeof jobSchema>>;
