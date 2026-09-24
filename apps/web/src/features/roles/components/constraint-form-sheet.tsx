'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { useAppForm } from '@/components/ui/tanstack-form';
import { usePermission } from '@/hooks/use-permission';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { createConstraintMutation, updateConstraintMutation } from '../api/mutations';
import { rolesQueryOptions } from '../api/queries';
import type { ConstraintPayload, SeparationConstraint } from '../api/types';

interface ConstraintFormSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  constraint?: SeparationConstraint;
}

const emptyValues: ConstraintPayload = {
  code: '',
  name: '',
  description: '',
  type: 'SSD',
  cardinality: 2,
  roleIds: [],
  isEnabled: true
};

export function ConstraintFormSheet({ open, onOpenChange, constraint }: ConstraintFormSheetProps) {
  const t = useTranslations('authorization.constraints');
  const tc = useTranslations('common');
  const { ts } = useSeedTranslation();
  const isEditing = Boolean(constraint);
  const canSave = usePermission(
    isEditing
      ? '/api/v1/admin/authorization/constraints/:id:PUT'
      : '/api/v1/admin/authorization/constraints:POST'
  );
  const { data: roles = [] } = useQuery({ ...rolesQueryOptions(), enabled: open });
  const createMutation = useMutation({
    ...mergeMutationOptions(createConstraintMutation, {
      onSuccess: () => {
        toast.success(t('messages.created'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.saveFailed'))
    })
  });
  const updateMutation = useMutation({
    ...mergeMutationOptions(updateConstraintMutation, {
      onSuccess: () => {
        toast.success(t('messages.updated'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.saveFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: emptyValues,
    onSubmit: async ({ value }) => {
      const payload = { ...value, cardinality: Number(value.cardinality) } as ConstraintPayload;
      if (constraint) {
        await updateMutation.mutateAsync({ id: constraint.id, data: payload });
      } else {
        await createMutation.mutateAsync(payload);
      }
    }
  });

  useEffect(() => {
    if (!open) return;
    form.reset(
      constraint
        ? {
            code: constraint.code,
            name: constraint.name,
            description: constraint.description,
            type: constraint.type,
            cardinality: constraint.cardinality,
            roleIds: constraint.roleIds,
            isEnabled: constraint.isEnabled
          }
        : emptyValues
    );
  }, [constraint, form, open]);

  const pending = createMutation.isPending || updateMutation.isPending;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col sm:max-w-xl'>
        <SheetHeader>
          <SheetTitle>{isEditing ? t('editTitle') : t('createTitle')}</SheetTitle>
          <SheetDescription>{t('formDescription')}</SheetDescription>
        </SheetHeader>
        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='constraint-form' className='space-y-4'>
              <form.TextField name='code' label={tc('code')} required placeholder='finance-ssd' />
              <form.TextField name='name' label={tc('name')} required />
              <form.TextareaField name='description' label={tc('description')} />
              <form.SelectField
                name='type'
                label={t('type')}
                required
                options={[
                  { value: 'SSD', label: t('ssd') },
                  { value: 'DSD', label: t('dsd') }
                ]}
              />
              <form.TextField name='cardinality' label={t('cardinality')} type='number' required />
              <form.SwitchField
                name='isEnabled'
                label={tc('enabled')}
                description={t('enabledDescription')}
              />
              <div className='space-y-2'>
                <Label>{t('roleSet')}</Label>
                <div className='grid gap-2 sm:grid-cols-2'>
                  {roles.map((role) => (
                    <form.Subscribe
                      key={role.id}
                      selector={(state) =>
                        ((state.values as ConstraintPayload).roleIds as number[]) ?? []
                      }
                    >
                      {(roleIds) => (
                        <label className='flex cursor-pointer items-center gap-2 rounded-md border p-2'>
                          <Checkbox
                            checked={roleIds.includes(role.id)}
                            onCheckedChange={(checked) =>
                              form.setFieldValue(
                                'roleIds',
                                checked
                                  ? [...roleIds, role.id]
                                  : roleIds.filter((id) => id !== role.id)
                              )
                            }
                          />
                          <span className='min-w-0 truncate text-sm'>{ts(role.name)}</span>
                        </label>
                      )}
                    </form.Subscribe>
                  ))}
                </div>
                <p className='text-xs text-muted-foreground'>{t('cardinalityHelp')}</p>
              </div>
            </form.Form>
          </form.AppForm>
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          {canSave && (
            <Button type='submit' form='constraint-form' isLoading={pending}>
              <Icons.check /> {tc('saveChanges')}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

export function ConstraintFormSheetTrigger() {
  const t = useTranslations('authorization.constraints');
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.add /> {t('add')}
      </Button>
      <ConstraintFormSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
