'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Icons } from '@/components/icons';
import { EmptyState } from '@/components/ui/empty-state';
import { deleteDictItemMutation, toggleDictItemMutation } from '../api/mutations';
import type { DictType, DictItem } from '../api/types';
import { DictItemFormSheet } from './dict-item-form-sheet';
import { AlertModal } from '@/components/modal/alert-modal';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface DictItemListProps {
  type: DictType;
  items: DictItem[];
  canCreate?: boolean;
  canUpdate?: boolean;
  canDelete?: boolean;
}

const COLOR_PREVIEW: Record<string, string> = {
  red: 'bg-red-500',
  green: 'bg-green-500',
  blue: 'bg-blue-500',
  orange: 'bg-orange-500',
  gray: 'bg-gray-400',
  purple: 'bg-purple-500',
  yellow: 'bg-yellow-500',
  pink: 'bg-pink-500'
};

export function DictItemList({
  type,
  items,
  canCreate = false,
  canUpdate = false,
  canDelete = false
}: DictItemListProps) {
  const t = useTranslations('dictionary');
  const tc = useTranslations('common');
  const [createOpen, setCreateOpen] = useState(false);
  const [editItem, setEditItem] = useState<DictItem | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [itemToDelete, setItemToDelete] = useState<DictItem | null>(null);

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteDictItemMutation, {
      onSuccess: () => {
        toast.success(t('messages.itemDeleted'));
        setDeleteOpen(false);
      },
      onError: () => toast.error(t('messages.itemDeleteFailed'))
    })
  });

  const toggleMutation = useMutation({
    ...mergeMutationOptions(toggleDictItemMutation, {
      onError: () => toast.error(t('messages.itemToggleFailed'))
    })
  });

  const handleDelete = (item: DictItem) => {
    setItemToDelete(item);
    setDeleteOpen(true);
  };

  const handleToggle = (item: DictItem, checked: boolean) => {
    toggleMutation.mutate({ id: item.id, enabled: checked });
  };

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() =>
          itemToDelete && deleteMutation.mutate({ id: itemToDelete.id, typeCode: type.code })
        }
        loading={deleteMutation.isPending}
      />
      <DictItemFormSheet typeCode={type.code} open={createOpen} onOpenChange={setCreateOpen} />
      <DictItemFormSheet
        typeCode={type.code}
        open={!!editItem}
        onOpenChange={(open) => {
          if (!open) setEditItem(null);
        }}
        existing={editItem}
      />

      <div className='flex h-full flex-col rounded-lg border bg-card'>
        {/* Header */}
        <div className='flex items-center justify-between border-b px-5 py-3'>
          <div>
            <div className='flex items-center gap-2'>
              <h3 className='text-sm font-semibold'>{type.name}</h3>
              {type.isSystem && (
                <Badge variant='secondary' className='text-[10px]'>
                  {t('type.sys')}
                </Badge>
              )}
            </div>
            <p className='text-muted-foreground text-xs mt-0.5'>
              <span className='font-mono'>{type.code}</span>
              {type.description && <span> &middot; {type.description}</span>}
              <span> &middot; {t('item.itemsCount', { count: items.length })}</span>
            </p>
          </div>
          {canCreate && (
            <Button size='sm' onClick={() => setCreateOpen(true)}>
              <Icons.add className='mr-1.5 h-3.5 w-3.5' /> {t('item.addItem')}
            </Button>
          )}
        </div>

        {/* Table */}
        <div className='flex-1 overflow-auto'>
          {items.length === 0 ? (
            <EmptyState
              icon={<Icons.code />}
              title={t('item.noItems')}
              description={t('item.createFirst')}
            />
          ) : (
            <table className='w-full text-sm'>
              <thead className='sticky top-0 bg-card'>
                <tr className='border-b'>
                  <th className='px-5 py-2.5 text-left text-xs font-medium text-muted-foreground'>
                    {t('item.label')}
                  </th>
                  <th className='px-5 py-2.5 text-left text-xs font-medium text-muted-foreground'>
                    {t('item.value')}
                  </th>
                  <th className='px-5 py-2.5 text-left text-xs font-medium text-muted-foreground'>
                    {t('item.appearance')}
                  </th>
                  <th className='px-5 py-2.5 text-center text-xs font-medium text-muted-foreground w-16'>
                    {t('item.order')}
                  </th>
                  <th className='px-5 py-2.5 text-center text-xs font-medium text-muted-foreground w-20'>
                    {tc('status')}
                  </th>
                  {(canUpdate || canDelete) && <th className='px-5 py-2.5 w-24' />}
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr
                    key={item.id}
                    className='group border-b last:border-b-0 transition-colors hover:bg-accent/30'
                  >
                    <td className='px-5 py-2.5'>
                      <div className='flex items-center gap-2'>
                        <span className='font-medium'>{item.label}</span>
                        {item.isDefault && (
                          <Badge variant='default' className='h-4 px-1 text-[10px]'>
                            {t('item.default')}
                          </Badge>
                        )}
                      </div>
                      {item.description && (
                        <p className='text-muted-foreground text-xs mt-0.5 max-w-xs truncate'>
                          {item.description}
                        </p>
                      )}
                    </td>
                    <td className='px-5 py-2.5'>
                      <code className='rounded bg-muted px-1.5 py-0.5 text-xs font-mono'>
                        {item.value}
                      </code>
                    </td>
                    <td className='px-5 py-2.5'>
                      <div className='flex items-center gap-3'>
                        {item.color && (
                          <div className='flex items-center gap-1.5'>
                            <span
                              className={`h-3 w-3 rounded-full border ${COLOR_PREVIEW[item.color] ?? ''}`}
                              style={
                                !COLOR_PREVIEW[item.color]
                                  ? { backgroundColor: item.color }
                                  : undefined
                              }
                            />
                            <span className='text-xs text-muted-foreground'>{item.color}</span>
                          </div>
                        )}
                        {item.icon && (
                          <div className='flex items-center gap-1'>
                            <Icons.code className='h-3 w-3 text-muted-foreground' />
                            <span className='text-xs text-muted-foreground'>{item.icon}</span>
                          </div>
                        )}
                        {!item.color && !item.icon && (
                          <span className='text-muted-foreground text-xs'>—</span>
                        )}
                      </div>
                    </td>
                    <td className='px-5 py-2.5 text-center'>
                      <span className='text-xs text-muted-foreground'>{item.sortOrder}</span>
                    </td>
                    <td className='px-5 py-2.5 text-center'>
                      <Switch
                        checked={item.isEnabled}
                        onCheckedChange={(checked) => handleToggle(item, checked)}
                        disabled={!canUpdate || toggleMutation.isPending}
                        className='scale-90'
                      />
                    </td>
                    {(canUpdate || canDelete) && (
                      <td className='px-5 py-2.5'>
                        <div className='flex items-center justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100'>
                          {canUpdate && (
                            <Button
                              size='icon'
                              variant='ghost'
                              className='h-7 w-7'
                              onClick={() => setEditItem(item)}
                            >
                              <Icons.edit className='h-3.5 w-3.5' />
                            </Button>
                          )}
                          {canDelete && (
                            <Button
                              size='icon'
                              variant='ghost'
                              className='h-7 w-7 hover:text-destructive'
                              onClick={() => handleDelete(item)}
                            >
                              <Icons.trash className='h-3.5 w-3.5' />
                            </Button>
                          )}
                        </div>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </>
  );
}
