'use client';

import { useSuspenseQuery } from '@tanstack/react-query';
import { assetStatsQueryOptions } from '../api/queries';
import { Icons } from '@/components/icons';
import { useDict } from '@/hooks/use-dict';
import { tagColorBg } from '@/lib/tag-color';
import { useFormat } from '@/hooks/use-format';
import { useTranslations } from 'next-intl';

export function AssetsStatsCards() {
  const { data } = useSuspenseQuery(assetStatsQueryOptions());
  const fmt = useFormat();
  const t = useTranslations('assets.stats');
  const dict = useDict();

  return (
    <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
      {/* Total count card */}
      <div className='rounded-lg border bg-card p-4'>
        <div className='mb-2 flex items-center justify-between'>
          <p className='text-muted-foreground text-sm'>{t('totalAssets')}</p>
          <Icons.workspace className='h-4 w-4 text-muted-foreground' />
        </div>
        <p className='text-2xl font-bold'>{fmt.number(data.totalCount)}</p>
        <p className='text-muted-foreground text-xs'>
          {data.totalSizeFormatted} {t('totalStorage')}
        </p>
      </div>

      {/* Category stats card */}
      <div className='rounded-lg border bg-card p-4'>
        <p className='mb-2 text-muted-foreground text-sm'>{t('byCategory')}</p>
        <div className='grid grid-cols-3 gap-2'>
          {Object.entries(data.categoryStats)
            .filter(([, count]) => (count as number) > 0)
            .map(([category, count]) => {
              const iconKey = dict.icon('asset_category', category) as
                | keyof typeof Icons
                | undefined;
              const IconComp = iconKey ? (Icons[iconKey] ?? Icons.page) : Icons.page;
              return (
                <div key={category} className='flex items-center gap-1.5'>
                  <IconComp className='h-3.5 w-3.5 text-muted-foreground' />
                  <div>
                    <p className='text-xs font-medium'>{String(count)}</p>
                    <p className='text-2xs text-muted-foreground uppercase'>
                      {dict.label('asset_category', category)}
                    </p>
                  </div>
                </div>
              );
            })}
        </div>
      </div>

      {/* Status stats card */}
      <div className='rounded-lg border bg-card p-4'>
        <p className='mb-2 text-muted-foreground text-sm'>{t('byStatus')}</p>
        <div className='space-y-2'>
          {Object.entries(data.statusStats)
            .filter(([, count]) => (count as number) > 0)
            .map(([status, count]) => {
              return (
                <div key={status} className='flex items-center justify-between'>
                  <div className='flex items-center gap-2'>
                    <span
                      className={`h-2 w-2 rounded-full ${tagColorBg(dict.color('asset_status', status)) ?? 'bg-tag-gray'}`}
                    />
                    <span className='text-xs'>{dict.label('asset_status', status)}</span>
                  </div>
                  <span className='text-xs font-medium'>{fmt.number(count as number)}</span>
                </div>
              );
            })}
        </div>
      </div>

      {/* Storage card */}
      <div className='rounded-lg border bg-card p-4'>
        <p className='mb-2 text-muted-foreground text-sm'>{t('storageUsage')}</p>
        <p className='text-2xl font-bold'>{data.totalSizeFormatted}</p>
        <p className='text-muted-foreground text-xs'>
          {t('filesStored', { count: data.totalCount })}
        </p>
      </div>
    </div>
  );
}
