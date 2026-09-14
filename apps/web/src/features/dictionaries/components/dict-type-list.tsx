'use client';

import { useMemo, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { deleteDictTypeMutation, toggleDictTypeMutation } from '../api/mutations';
import type { DictType } from '../api/types';
import { DictTypeFormSheet } from './dict-type-form-sheet';
import { AlertModal } from '@/components/modal/alert-modal';
import { cn } from '@/lib/utils';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface DictTypeListProps {
  types: DictType[];
  selectedType: DictType | null;
  onSelect: (type: DictType | null) => void;
  canCreate?: boolean;
  canUpdate?: boolean;
  canDelete?: boolean;
}

const PAGE_SIZE = 10;

export function DictTypeList({
  types,
  selectedType,
  onSelect,
  canCreate = false,
  canUpdate = false,
  canDelete = false
}: DictTypeListProps) {
  const t = useTranslations('dictionary');
  const [createOpen, setCreateOpen] = useState(false);
  const [editType, setEditType] = useState<DictType | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [typeToDelete, setTypeToDelete] = useState<DictType | null>(null);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);

  const filteredTypes = useMemo(() => {
    if (!search.trim()) return types;
    const q = search.toLowerCase();
    return types.filter(
      (t) => t.name.toLowerCase().includes(q) || t.code.toLowerCase().includes(q)
    );
  }, [types, search]);

  const totalPages = Math.ceil(filteredTypes.length / PAGE_SIZE);
  const pagedTypes = useMemo(
    () => filteredTypes.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE),
    [filteredTypes, page]
  );

  // Reset to page 1 when search changes
  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteDictTypeMutation, {
      onSuccess: () => {
        toast.success(t('messages.typeDeleted'));
        setDeleteOpen(false);
        if (typeToDelete && typeToDelete.id === selectedType?.id) {
          const remaining = types.filter((t) => t.id !== typeToDelete.id);
          onSelect(remaining.length > 0 ? remaining[0] : null);
        }
      },
      onError: () => toast.error(t('messages.typeDeleteFailed'))
    })
  });

  const toggleMutation = useMutation({
    ...mergeMutationOptions(toggleDictTypeMutation, {
      onError: () => toast.error(t('messages.typeToggleFailed'))
    })
  });

  const handleDelete = (type: DictType) => {
    if (type.isSystem) {
      toast.error(t('messages.systemTypeCannotDelete'));
      return;
    }
    setTypeToDelete(type);
    setDeleteOpen(true);
  };

  const handleToggle = (type: DictType, checked: boolean) => {
    toggleMutation.mutate({ id: type.id, enabled: checked });
  };

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => typeToDelete && deleteMutation.mutate(typeToDelete.id)}
        loading={deleteMutation.isPending}
      />
      <DictTypeFormSheet open={createOpen} onOpenChange={setCreateOpen} />
      <DictTypeFormSheet
        open={!!editType}
        onOpenChange={(open) => {
          if (!open) setEditType(null);
        }}
        existing={editType}
      />

      <div className='flex h-full flex-col rounded-lg border bg-card'>
        {/* Header */}
        <div className='flex items-center justify-between border-b px-4 py-3'>
          <div>
            <h3 className='text-sm font-semibold'>{t('type.types')}</h3>
            <p className='text-muted-foreground text-xs'>
              {t('type.total', { count: types.length })}
            </p>
          </div>
          {canCreate && (
            <Button
              size='icon'
              variant='outline'
              className='h-7 w-7'
              onClick={() => setCreateOpen(true)}
            >
              <Icons.add className='h-3.5 w-3.5' />
            </Button>
          )}
        </div>

        {/* Search */}
        <div className='border-b px-3 py-2'>
          <div className='relative'>
            <Icons.search className='absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground' />
            <Input
              placeholder={t('type.searchTypes')}
              value={search}
              onChange={(e) => handleSearch(e.target.value)}
              className='h-8 pl-8 text-xs'
            />
          </div>
        </div>

        {/* List */}
        <div className='flex-1 overflow-auto'>
          {pagedTypes.length === 0 ? (
            <div className='py-8 text-center text-muted-foreground text-xs'>
              {search ? t('type.noMatching') : t('type.noTypes')}
            </div>
          ) : (
            pagedTypes.map((type) => (
              <div
                key={type.id}
                role='button'
                tabIndex={0}
                className={cn(
                  'group flex w-full cursor-pointer items-center justify-between border-b px-4 py-2.5 text-left transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring last:border-b-0',
                  selectedType?.id === type.id && 'bg-accent'
                )}
                onClick={() => onSelect(type)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    onSelect(type);
                  }
                }}
              >
                <div className='flex flex-col gap-0.5 min-w-0'>
                  <div className='flex items-center gap-1.5'>
                    <span className='text-sm font-medium truncate'>{type.name}</span>
                    {type.isSystem && (
                      <Badge variant='secondary' className='h-4 shrink-0 px-1 text-[10px]'>
                        {t('type.sys')}
                      </Badge>
                    )}
                    {!type.isEnabled && (
                      <Badge variant='outline' className='h-4 shrink-0 px-1 text-[10px]'>
                        {t('type.off')}
                      </Badge>
                    )}
                  </div>
                  <span className='text-muted-foreground text-[11px] font-mono truncate'>
                    {type.code}
                  </span>
                </div>

                {/* Actions — show on hover/selected */}
                {(canUpdate || canDelete) && (
                  <div
                    className={cn(
                      'flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100',
                      selectedType?.id === type.id && 'opacity-100'
                    )}
                  >
                    {canUpdate && (
                      <>
                        <Switch
                          checked={type.isEnabled}
                          onCheckedChange={(checked) => handleToggle(type, checked)}
                          onClick={(e) => e.stopPropagation()}
                          onKeyDown={(e) => e.stopPropagation()}
                          disabled={toggleMutation.isPending}
                          className='scale-75'
                        />
                        <Button
                          size='icon'
                          variant='ghost'
                          className='h-6 w-6'
                          onClick={(e) => {
                            e.stopPropagation();
                            setEditType(type);
                          }}
                        >
                          <Icons.edit className='h-3 w-3' />
                        </Button>
                      </>
                    )}
                    {canDelete && (
                      <Button
                        size='icon'
                        variant='ghost'
                        className='h-6 w-6'
                        disabled={type.isSystem}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(type);
                        }}
                      >
                        <Icons.trash className='h-3 w-3' />
                      </Button>
                    )}
                  </div>
                )}
              </div>
            ))
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className='flex items-center justify-between border-t px-3 py-2'>
            <span className='text-muted-foreground text-[11px]'>
              {page}/{totalPages}
            </span>
            <div className='flex items-center gap-1'>
              <Button
                size='icon'
                variant='ghost'
                className='h-6 w-6'
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <Icons.chevronLeft className='h-3.5 w-3.5' />
              </Button>
              <Button
                size='icon'
                variant='ghost'
                className='h-6 w-6'
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                <Icons.chevronRight className='h-3.5 w-3.5' />
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  );
}
