'use client';

import { useState } from 'react';
import { useAppForm } from '@/components/ui/tanstack-form';
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
import { Icons } from '@/components/icons';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { createResourceMutation, updateResourceMutation } from '../api/mutations';
import type { Resource } from '../api/types';
import { toast } from 'sonner';
import {
  getCreateResourceSchema,
  getUpdateResourceSchema,
  type CreateResourceFormValues,
  type UpdateResourceFormValues
} from '../schemas/resource';
import { mergeMutationOptions } from '@/lib/mutation-utils';

const ACTION_OPTIONS = [
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'PATCH', label: 'PATCH' },
  { value: 'DELETE', label: 'DELETE' }
];

interface ResourceFormSheetProps {
  resource?: Resource;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ResourceFormSheet({ resource, open, onOpenChange }: ResourceFormSheetProps) {
  const t = useTranslations('resources');
  const tc = useTranslations('common');
  const isEdit = !!resource;
  const validationMessages = {
    codeMin: t('validation.codeMin'),
    nameMin: t('validation.nameMin'),
    pathRequired: t('validation.pathRequired'),
    actionsRequired: t('validation.actionsRequired'),
    categoryRequired: t('validation.categoryRequired')
  };

  const categoryOptions = [
    { value: 'admin', label: t('form.admin') },
    { value: 'user', label: t('form.user') }
  ];

  const createMutation = useMutation({
    ...mergeMutationOptions(createResourceMutation, {
      onSuccess: () => {
        toast.success(t('messages.createSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.createFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateResourceMutation, {
      onSuccess: () => {
        toast.success(t('messages.updateSuccess'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.updateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: isEdit
      ? ({
          name: resource.name ?? '',
          path: resource.path ?? '',
          actions: resource.actions ?? [],
          category: resource.category ?? 'admin',
          description: resource.description ?? '',
          module: resource.module ?? '',
          sortOrder: resource.sortOrder ?? 0,
          isEnabled: resource.isEnabled ?? true
        } as UpdateResourceFormValues)
      : ({
          code: '',
          name: '',
          path: '',
          actions: ['GET'],
          category: 'admin',
          description: '',
          module: '',
          sortOrder: 0
        } as CreateResourceFormValues),
    validators: {
      onSubmit: isEdit
        ? getUpdateResourceSchema(validationMessages)
        : getCreateResourceSchema(validationMessages)
    },
    onSubmit: async ({ value }) => {
      if (isEdit) {
        const v = value as UpdateResourceFormValues;
        await updateMutation.mutateAsync({
          id: resource!.id,
          values: {
            name: v.name,
            path: v.path,
            actions: v.actions,
            category: v.category,
            description: v.description || undefined,
            module: v.module || undefined,
            sortOrder: v.sortOrder,
            isEnabled: v.isEnabled
          }
        });
      } else {
        const v = value as CreateResourceFormValues;
        await createMutation.mutateAsync({
          code: v.code,
          name: v.name,
          path: v.path,
          actions: v.actions,
          category: v.category,
          description: v.description || undefined,
          module: v.module || undefined,
          sortOrder: v.sortOrder
        });
      }
    }
  });

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEdit ? t('form.editTitle') : t('form.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEdit ? t('form.editDescription') : t('form.newDescription')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='resource-form-sheet' className='space-y-4'>
              {!isEdit && (
                <form.TextField
                  name='code'
                  label={t('form.code')}
                  required
                  placeholder={t('form.codePlaceholder')}
                />
              )}
              <form.TextField
                name='name'
                label={tc('name')}
                required
                placeholder={t('form.namePlaceholder')}
              />
              <form.TextField
                name='path'
                label={t('form.apiPath')}
                required
                placeholder='/api/v1/admin/users/:id'
                description={t('form.pathHelp')}
              />

              <div className='space-y-2'>
                <Label>{t('form.actions')}</Label>
                <div className='grid grid-cols-2 gap-2'>
                  {ACTION_OPTIONS.map((option) => (
                    <form.Subscribe
                      key={option.value}
                      selector={(state) =>
                        ((state.values as Record<string, unknown>).actions as
                          | string[]
                          | undefined) ?? []
                      }
                    >
                      {(actions) => (
                        <div className='flex items-center gap-2'>
                          <Checkbox
                            id={`action-${option.value}`}
                            checked={actions.includes(option.value)}
                            onCheckedChange={(checked) => {
                              if (checked) {
                                form.setFieldValue('actions', [...actions, option.value]);
                              } else {
                                form.setFieldValue(
                                  'actions',
                                  actions.filter((a) => a !== option.value)
                                );
                              }
                            }}
                          />
                          <label
                            htmlFor={`action-${option.value}`}
                            className='cursor-pointer text-sm leading-none'
                          >
                            {option.label}
                          </label>
                        </div>
                      )}
                    </form.Subscribe>
                  ))}
                </div>
              </div>

              <form.SelectField
                name='category'
                label={t('form.category')}
                required
                options={categoryOptions}
                placeholder={t('form.selectType')}
              />
              <form.TextField
                name='module'
                label={t('form.module')}
                placeholder={t('form.categoryPlaceholder')}
              />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('form.describePlaceholder')}
              />
              <form.TextField
                name='sortOrder'
                label={t('form.sortOrder')}
                type='number'
                placeholder='0'
              />

              {isEdit && (
                <form.SwitchField
                  name='isEnabled'
                  label={tc('enabled')}
                  description={t('form.toggleEnabled')}
                />
              )}
            </form.Form>
          </form.AppForm>
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='resource-form-sheet' isLoading={isPending}>
            <Icons.check /> {isEdit ? t('form.updateResource') : t('form.createResource')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// Trigger Button
// ============================================================

export function ResourceFormSheetTrigger() {
  const t = useTranslations('resources');
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.add className='mr-2 h-4 w-4' /> {t('list.addResource')}
      </Button>
      <ResourceFormSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
