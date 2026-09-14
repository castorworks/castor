'use client';

import { useStore } from '@tanstack/react-form';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { FieldDescription, FieldLabel } from '@/components/ui/field';
import {
  useFieldContext,
  FormFieldSet,
  FormField,
  FormFieldError,
  createFormField
} from '@/components/ui/form-context';
import { Icons } from '@/components/icons';

interface DateTimeFieldProps {
  label: string;
  description?: string;
  clearLabel?: string;
}

function DateTimeField({ label, description, clearLabel }: DateTimeFieldProps) {
  const field = useFieldContext();
  const value = useStore(field.store, (s) => s.value) as string | null;

  // Convert ISO 8601 string to datetime-local format (YYYY-MM-DDTHH:mm)
  const inputValue = value ? toDateTimeLocalValue(value) : '';

  return (
    <FormFieldSet>
      <FormField>
        <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
        <div className='flex items-center gap-2'>
          <Input
            id={field.name}
            type='datetime-local'
            value={inputValue}
            onBlur={field.handleBlur}
            onChange={(e) => {
              const val = e.target.value;
              // Convert datetime-local value to ISO 8601 string or null
              field.handleChange(val ? new Date(val).toISOString() : null);
            }}
            className='flex-1'
          />
          {value && (
            <Button
              type='button'
              variant='ghost'
              size='icon'
              onClick={() => field.handleChange(null)}
              title={clearLabel}
            >
              <Icons.close className='h-4 w-4' />
            </Button>
          )}
        </div>
        {description && <FieldDescription>{description}</FieldDescription>}
      </FormField>
      <FormFieldError />
    </FormFieldSet>
  );
}

/** Convert an ISO 8601 string to the format expected by datetime-local input */
function toDateTimeLocalValue(iso: string): string {
  try {
    const date = new Date(iso);
    if (isNaN(date.getTime())) return '';
    // Format: YYYY-MM-DDTHH:mm
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  } catch {
    return '';
  }
}

export const FormDateTimeField = createFormField(DateTimeField);
