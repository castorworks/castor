'use client';
import { AlertModal } from '@/components/modal/alert-modal';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { deleteAssetMutation, updateAssetStatusMutation } from '../../api/mutations';
import type { Asset, AssetStatus } from '../../api/types';
import { Icons } from '@/components/icons';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { AssetFormSheet } from '../asset-form-sheet';
import { getAssetDownloadUrl } from '../../api/service';
import { getDictLabel } from '@/lib/dict';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';

interface CellActionProps {
  data: Asset;
}

export function CellAction({ data }: CellActionProps) {
  const t = useTranslations('assets');
  const tc = useTranslations('common');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const canDownload = usePermission('/api/v1/admin/assets/:id:GET');
  const canUpdateAsset = usePermission('/api/v1/admin/assets/:id:PUT');
  const canUpdateAssetStatus = usePermission('/api/v1/admin/assets/:id/status:PUT');
  const canDeleteAsset = usePermission('/api/v1/admin/assets/:id:DELETE');

  const statusOptions = [
    { value: 'ACTIVE' as AssetStatus, label: getDictLabel('asset_status', 'ACTIVE') },
    { value: 'PENDING' as AssetStatus, label: getDictLabel('asset_status', 'PENDING') },
    { value: 'ARCHIVED' as AssetStatus, label: getDictLabel('asset_status', 'ARCHIVED') },
    { value: 'DELETED' as AssetStatus, label: getDictLabel('asset_status', 'DELETED') }
  ];

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteAssetMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => {
        toast.error(t('messages.deleteFailed'));
      }
    })
  });

  const statusMutation = useMutation({
    ...mergeMutationOptions(updateAssetStatusMutation, {
      onSuccess: () => {
        toast.success(t('messages.statusUpdated'));
      },
      onError: () => {
        toast.error(t('messages.statusUpdateFailed'));
      }
    })
  });

  const downloadUrl = getAssetDownloadUrl(data.objectKey);

  if (!canDownload && !canUpdateAsset && !canUpdateAssetStatus && !canDeleteAsset) {
    return null;
  }

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate(data.id)}
        loading={deleteMutation.isPending}
      />
      <AssetFormSheet asset={data} open={editOpen} onOpenChange={setEditOpen} />
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant='ghost' className='h-8 w-8 p-0'>
            <span className='sr-only'>{tc('openMenu')}</span>
            <Icons.ellipsis className='h-4 w-4' />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end'>
          <DropdownMenuLabel>{tc('actions')}</DropdownMenuLabel>
          {canUpdateAsset && (
            <DropdownMenuItem onClick={() => setEditOpen(true)}>
              <Icons.edit className='mr-2 h-4 w-4' /> {t('table.editMetadata')}
            </DropdownMenuItem>
          )}
          {canDownload && (
            <DropdownMenuItem asChild>
              <a href={downloadUrl} download>
                <Icons.download className='mr-2 h-4 w-4' /> {tc('download')}
              </a>
            </DropdownMenuItem>
          )}
          {(canUpdateAssetStatus || canDeleteAsset) && (
            <>
              <DropdownMenuSeparator />
              {canUpdateAssetStatus && (
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <Icons.settings className='mr-2 h-4 w-4' /> {t('table.changeStatus')}
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent>
                    {statusOptions.map((option) => (
                      <DropdownMenuItem
                        key={option.value}
                        onClick={() =>
                          statusMutation.mutate({ id: data.id, values: { status: option.value } })
                        }
                      >
                        {option.label}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              )}
              {canUpdateAssetStatus && canDeleteAsset && <DropdownMenuSeparator />}
              {canDeleteAsset && (
                <DropdownMenuItem onClick={() => setDeleteOpen(true)}>
                  <Icons.trash className='mr-2 h-4 w-4' /> {tc('delete')}
                </DropdownMenuItem>
              )}
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
