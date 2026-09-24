'use client';

import { useAppForm } from '@/components/ui/tanstack-form';
import { Button } from '@/components/ui/button';
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
import { createSettingMutation } from '../../api/mutations';
import { toast } from 'sonner';
import { createSettingSchema, type CreateSettingFormValues } from '../../schemas/setting';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useState } from 'react';

const TYPE_OPTIONS = [
  { value: 'STRING', label: 'String' },
  { value: 'NUMBER', label: 'Number' },
  { value: 'BOOL', label: 'Boolean' },
  { value: 'JSON', label: 'JSON' },
  { value: 'ARRAY', label: 'Array' },
  { value: 'SECRET', label: 'Secret' }
];

interface SettingFormSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SettingFormSheet({ open, onOpenChange }: SettingFormSheetProps) {
  const t = useTranslations('settings');
  const tc = useTranslations('common');
  const createMutation = useMutation({
    ...mergeMutationOptions(createSettingMutation, {
      onSuccess: () => {
        toast.success(t('messages.createSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.createFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: {
      key: '',
      name: '',
      value: '',
      type: 'STRING',
      category: 'GENERAL',
      description: '',
      defaultVal: '',
      isPublic: false,
      sortOrder: 0
    } as CreateSettingFormValues,
    validators: {
      onSubmit: createSettingSchema
    },
    onSubmit: async ({ value }) => {
      const v = value as CreateSettingFormValues;
      await createMutation.mutateAsync({
        key: v.key,
        name: v.name,
        value: v.value || undefined,
        type: v.type || 'STRING',
        category: v.category || undefined,
        description: v.description || undefined,
        defaultVal: v.defaultVal || undefined,
        isPublic: v.isPublic ?? false,
        sortOrder: v.sortOrder ?? 0
      });
    }
  });

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{t('form.newTitle')}</SheetTitle>
          <SheetDescription>{t('form.newDescription')}</SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='setting-form-sheet' className='space-y-4'>
              <form.TextField name='key' label={tc('key')} required placeholder='site_name' />
              <form.TextField
                name='name'
                label={tc('name')}
                required
                placeholder={t('form.siteName')}
              />
              <form.SelectField
                name='type'
                label={tc('type')}
                required
                options={TYPE_OPTIONS}
                placeholder={t('form.selectType')}
              />
              <form.TextField name='category' label={tc('category')} placeholder='GENERAL' />
              <form.TextField
                name='value'
                label={tc('value')}
                placeholder={t('form.initialValue')}
              />
              <form.TextField
                name='defaultVal'
                label={t('form.defaultValue')}
                placeholder={t('form.defaultValue')}
              />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('form.describe')}
              />
              <form.SwitchField
                name='isPublic'
                label={tc('public')}
                description={t('form.publicAccess')}
              />
              <form.TextField
                name='sortOrder'
                label={t('form.sortOrder')}
                type='number'
                placeholder='0'
              />
            </form.Form>
          </form.AppForm>
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='setting-form-sheet' isLoading={createMutation.isPending}>
            <Icons.check /> {t('form.createSetting')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// Trigger Button
// ============================================================

export function SettingFormSheetTrigger() {
  const t = useTranslations('settings');
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.add className='mr-2 h-4 w-4' /> {t('table.addSetting')}
      </Button>
      <SettingFormSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
