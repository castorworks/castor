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
import { updateAssetMutation } from '../api/mutations';
import type { Asset } from '../api/types';
import { toast } from 'sonner';
import { updateAssetSchema, type UpdateAssetFormValues } from '../schemas/asset';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { AssetReferences } from './asset-references';
import { usePermission } from '@/hooks/use-permission';

interface AssetFormSheetProps {
  asset: Asset;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AssetFormSheet({ asset, open, onOpenChange }: AssetFormSheetProps) {
  const t = useTranslations('assets');
  const tc = useTranslations('common');
  const canViewDetail = usePermission('/api/v1/admin/assets/:id:GET');
  const updateMutation = useMutation({
    ...mergeMutationOptions(updateAssetMutation, {
      onSuccess: () => {
        toast.success(t('messages.updateSuccess'));
        onOpenChange(false);
      },
      // 后端消息已本地化（如“资产正在被使用”），比通用失败提示更有用。
      onError: (error) => toast.error(error.message || t('messages.updateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: {
      name: asset.name ?? '',
      description: asset.description ?? '',
      tags: asset.tags ?? '',
      folderPath: asset.folderPath ?? '',
      isPublic: asset.isPublic ?? false
    } as UpdateAssetFormValues,
    validators: {
      onSubmit: updateAssetSchema
    },
    onSubmit: async ({ value }) => {
      const v = value as UpdateAssetFormValues;
      await updateMutation.mutateAsync({
        id: asset.id,
        values: {
          name: v.name || undefined,
          description: v.description || undefined,
          tags: v.tags || undefined,
          folderPath: v.folderPath || undefined,
          isPublic: v.isPublic
        }
      });
    }
  });

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{t('form.editTitle')}</SheetTitle>
          <SheetDescription>
            {t('form.updateDescription')} {asset.name}.
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='asset-form-sheet' className='space-y-4'>
              <form.TextField name='name' label={tc('name')} placeholder={t('form.assetName')} />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('form.describe')}
              />
              <form.TextField name='tags' label={tc('tags')} placeholder={t('form.tags')} />
              <form.TextField
                name='folderPath'
                label={t('form.folderPath')}
                placeholder='/path/to/folder'
              />
              <form.SwitchField
                name='isPublic'
                label={tc('public')}
                description={t('form.publicAccess')}
              />
            </form.Form>
          </form.AppForm>
          {canViewDetail && (
            <div className='mt-6'>
              <AssetReferences assetId={asset.id} enabled={open} />
            </div>
          )}
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='asset-form-sheet' isLoading={updateMutation.isPending}>
            <Icons.check /> {tc('saveChanges')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
