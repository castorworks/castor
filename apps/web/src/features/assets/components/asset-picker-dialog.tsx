'use client';

import { useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { useDebounce } from '@/hooks/use-debounce';
import { useDict } from '@/hooks/use-dict';
import { cn } from '@/lib/utils';
import { assetsQueryOptions } from '../api/queries';
import type { Asset, AssetCategory } from '../api/types';
import { AssetThumbnail } from './asset-thumbnail';

const PAGE_SIZE = 12;

interface AssetPickerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (asset: Asset) => void;
  /** 只列某一分类，如头像、封面只要 IMAGE。 */
  category?: AssetCategory;
  /** 当前已选的 objectKey，用于高亮。 */
  selectedKey?: string;
}

/** 从资产库挑选一个已有文件。只列资产库（LIBRARY）中生效的资产。 */
export function AssetPickerDialog({
  open,
  onOpenChange,
  onSelect,
  category,
  selectedKey
}: AssetPickerDialogProps) {
  const t = useTranslations('assets.picker');
  const tp = useTranslations('pagination');
  const dict = useDict();
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const keyword = useDebounce(search, 300);

  const { data, isPending, isError } = useQuery({
    ...assetsQueryOptions({
      page,
      pageSize: PAGE_SIZE,
      scope: 'LIBRARY',
      status: 'ACTIVE',
      ...(category && { category }),
      ...(keyword && { search: keyword })
    }),
    enabled: open,
    placeholderData: keepPreviousData
  });

  const assets = data?.list ?? [];
  const totalPages = Math.max(data?.totalPages ?? 1, 1);

  // 关闭时清空关键词与页码，下次（可能是另一个字段）从头开始。
  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setSearch('');
      setPage(1);
    }
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className='sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>

        <Input
          value={search}
          placeholder={t('search')}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
        />

        <div className='min-h-64'>
          {isPending ? (
            <p className='text-muted-foreground py-10 text-center text-sm'>{t('loading')}</p>
          ) : isError ? (
            <p className='text-destructive py-10 text-center text-sm'>{t('loadFailed')}</p>
          ) : assets.length === 0 ? (
            <p className='text-muted-foreground py-10 text-center text-sm'>{t('empty')}</p>
          ) : (
            <ul className='grid grid-cols-2 gap-2 sm:grid-cols-3'>
              {assets.map((asset) => {
                const iconKey = dict.icon('asset_category', asset.category) as
                  | keyof typeof Icons
                  | undefined;
                const Icon = iconKey ? (Icons[iconKey] ?? Icons.page) : Icons.page;
                return (
                  <li key={asset.id}>
                    <button
                      type='button'
                      onClick={() => {
                        onSelect(asset);
                        handleOpenChange(false);
                      }}
                      className={cn(
                        'flex w-full items-center gap-2 rounded-lg border p-2 text-left transition-colors hover:bg-accent',
                        asset.objectKey === selectedKey && 'border-primary bg-primary/5'
                      )}
                    >
                      <AssetThumbnail
                        objectKey={asset.objectKey}
                        isImage={asset.category === 'IMAGE'}
                        icon={Icon}
                      />
                      <span className='min-w-0'>
                        <span className='block truncate font-medium text-sm'>{asset.name}</span>
                        <span className='text-muted-foreground block truncate text-xs'>
                          {asset.sizeFormatted} &middot; {asset.extension.toUpperCase()}
                        </span>
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <DialogFooter className='items-center sm:justify-between'>
          <span className='text-muted-foreground text-xs'>
            {tp('pageOf', { current: page, total: totalPages })}
          </span>
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              {tp('previous')}
            </Button>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              {tp('next')}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
