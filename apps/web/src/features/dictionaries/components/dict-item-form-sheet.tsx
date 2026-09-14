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
import { createDictItemMutation, updateDictItemMutation } from '../api/mutations';
import { toast } from 'sonner';
import { createDictItemSchema, type CreateDictItemFormValues } from '../schemas/dictionary';
import type { DictItem } from '../api/types';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface DictItemFormSheetProps {
  typeCode: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existing?: DictItem | null;
}

export function DictItemFormSheet({
  typeCode,
  open,
  onOpenChange,
  existing
}: DictItemFormSheetProps) {
  const t = useTranslations('dictionary');
  const tc = useTranslations('common');
  const isEdit = !!existing;

  const createMutation = useMutation({
    ...mergeMutationOptions(createDictItemMutation, {
      onSuccess: () => {
        toast.success(t('messages.itemCreated'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.itemCreateFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateDictItemMutation, {
      onSuccess: () => {
        toast.success(t('messages.itemUpdated'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.itemUpdateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: {
      label: existing?.label ?? '',
      value: existing?.value ?? '',
      description: existing?.description ?? '',
      color: existing?.color ?? '',
      icon: existing?.icon ?? '',
      isDefault: existing?.isDefault ?? false,
      sortOrder: existing?.sortOrder ?? 0
    } as CreateDictItemFormValues,
    validators: {
      onSubmit: createDictItemSchema
    },
    onSubmit: async ({ value }) => {
      const v = value as CreateDictItemFormValues;
      if (isEdit && existing) {
        await updateMutation.mutateAsync({
          id: existing.id,
          values: {
            label: v.label,
            value: v.value,
            description: v.description || undefined,
            color: v.color || undefined,
            icon: v.icon || undefined,
            isDefault: v.isDefault ?? false,
            sortOrder: v.sortOrder ?? 0
          }
        });
      } else {
        await createMutation.mutateAsync({
          typeCode,
          label: v.label,
          value: v.value,
          description: v.description || undefined,
          color: v.color || undefined,
          icon: v.icon || undefined,
          isDefault: v.isDefault ?? false,
          sortOrder: v.sortOrder ?? 0
        });
      }
    }
  });

  return (
    <Sheet key={existing?.id ?? 'create'} open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEdit ? t('item.editTitle') : t('item.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEdit
              ? t('item.editExisting', { label: existing?.label })
              : t('item.newDescription', { typeCode })}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='dict-item-form-sheet' className='space-y-4'>
              <form.TextField
                name='label'
                label={t('item.label')}
                required
                placeholder={t('fields.itemLabelPlaceholder')}
              />
              <form.TextField
                name='value'
                label={t('item.value')}
                required
                placeholder={t('fields.itemValuePlaceholder')}
              />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('fields.describeItem')}
              />
              <form.TextField
                name='color'
                label={t('fields.color')}
                placeholder={t('fields.colorPlaceholder')}
              />
              <form.TextField
                name='icon'
                label={t('fields.icon')}
                placeholder={t('item.iconIdentifier')}
              />
              <form.SwitchField
                name='isDefault'
                label={t('fields.default')}
                description={t('item.defaultOption')}
              />
              <form.TextField
                name='sortOrder'
                label={t('fields.sortOrder')}
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
          <Button
            type='submit'
            form='dict-item-form-sheet'
            isLoading={createMutation.isPending || updateMutation.isPending}
          >
            <Icons.check /> {isEdit ? t('item.updateItem') : t('item.createItem')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
