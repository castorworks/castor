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
import { createDictTypeMutation, updateDictTypeMutation } from '../api/mutations';
import { toast } from 'sonner';
import { createDictTypeSchema, type CreateDictTypeFormValues } from '../schemas/dictionary';
import type { DictType } from '../api/types';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface DictTypeFormSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existing?: DictType | null;
}

export function DictTypeFormSheet({ open, onOpenChange, existing }: DictTypeFormSheetProps) {
  const t = useTranslations('dictionary');
  const tc = useTranslations('common');
  const isEdit = !!existing;

  const createMutation = useMutation({
    ...mergeMutationOptions(createDictTypeMutation, {
      onSuccess: () => {
        toast.success(t('messages.typeCreated'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.typeCreateFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateDictTypeMutation, {
      onSuccess: () => {
        toast.success(t('messages.typeUpdated'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.typeUpdateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: {
      code: existing?.code ?? '',
      name: existing?.name ?? '',
      description: existing?.description ?? '',
      sortOrder: existing?.sortOrder ?? 0
    } as CreateDictTypeFormValues,
    validators: {
      onSubmit: createDictTypeSchema
    },
    onSubmit: async ({ value }) => {
      const v = value as CreateDictTypeFormValues;
      if (isEdit && existing) {
        await updateMutation.mutateAsync({
          id: existing.id,
          values: {
            name: v.name,
            description: v.description || undefined,
            sortOrder: v.sortOrder ?? 0
          }
        });
      } else {
        await createMutation.mutateAsync({
          code: v.code,
          name: v.name,
          description: v.description || undefined,
          sortOrder: v.sortOrder ?? 0
        });
      }
    }
  });

  return (
    <Sheet key={existing?.id ?? 'create'} open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEdit ? t('type.editTitle') : t('type.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEdit ? t('type.editExisting', { name: existing?.name }) : t('type.newDescription')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='dict-type-form-sheet' className='space-y-4'>
              <form.TextField
                name='code'
                label={tc('code')}
                required
                placeholder={t('fields.codePlaceholder')}
                disabled={isEdit}
              />
              <form.TextField
                name='name'
                label={tc('name')}
                required
                placeholder={t('fields.namePlaceholder')}
              />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('fields.describeType')}
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
            form='dict-type-form-sheet'
            isLoading={createMutation.isPending || updateMutation.isPending}
          >
            <Icons.check /> {isEdit ? t('type.updateType') : t('type.createType')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
