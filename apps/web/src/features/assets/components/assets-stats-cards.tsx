'use client';

import { useSuspenseQuery } from '@tanstack/react-query';
import { assetStatsQueryOptions } from '../api/queries';
import { Icons } from '@/components/icons';
import { getDictIcon, getDictLabel, getDictItems } from '@/lib/dict';
import { useFormat } from '@/hooks/use-format';
import { useTranslations } from 'next-intl';

const COLOR_DOT_MAP: Record<string, string> = {
  green: 'bg-green-500',
  orange: 'bg-yellow-500',
  gray: 'bg-gray-500',
  red: 'bg-red-500',
  blue: 'bg-blue-500'
};

function getStatusDot(status: string): string {
  const items = getDictItems('asset_status');
  const item = items.find((i) => i.value === status);
  return COLOR_DOT_MAP[item?.color ?? ''] ?? 'bg-gray-500';
}

export function AssetsStatsCards() {
  const { data } = useSuspenseQuery(assetStatsQueryOptions());
  const fmt = useFormat();
  const t = useTranslations('assets.stats');

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
              const iconKey = getDictIcon('asset_category', category) as
                | keyof typeof Icons
                | undefined;
              const IconComp = iconKey ? (Icons[iconKey] ?? Icons.page) : Icons.page;
              return (
                <div key={category} className='flex items-center gap-1.5'>
                  <IconComp className='h-3.5 w-3.5 text-muted-foreground' />
                  <div>
                    <p className='text-xs font-medium'>{String(count)}</p>
                    <p className='text-[10px] text-muted-foreground uppercase'>
                      {getDictLabel('asset_category', category)}
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
                    <span className={`h-2 w-2 rounded-full ${getStatusDot(status)}`} />
                    <span className='text-xs'>{getDictLabel('asset_status', status)}</span>
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
          {fmt.number(data.totalCount)} {t('filesStored')}
        </p>
      </div>
    </div>
  );
}
