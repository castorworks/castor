'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { Icons } from '@/components/icons';
import { assetDetailQueryOptions } from '../api/queries';

interface AssetReferencesProps {
  assetId: number;
  /** 只在面板打开时才请求详情。 */
  enabled: boolean;
}

/** “被谁使用”：列出引用该资产的业务记录，解释它为何不能删除、下线或取消公开。 */
export function AssetReferences({ assetId, enabled }: AssetReferencesProps) {
  const t = useTranslations('assets.references');
  const { data, isPending } = useQuery({ ...assetDetailQueryOptions(assetId), enabled });

  const references = data?.references ?? [];

  return (
    <section className='space-y-2 rounded-lg border p-3'>
      <div className='flex items-center gap-2'>
        <Icons.paperclip className='h-4 w-4 text-muted-foreground' />
        <h3 className='font-medium text-sm'>{t('title')}</h3>
        {references.length > 0 && <Badge variant='secondary'>{references.length}</Badge>}
      </div>
      {isPending && enabled ? (
        <p className='text-muted-foreground text-xs'>{t('loading')}</p>
      ) : references.length === 0 ? (
        <p className='text-muted-foreground text-xs'>{t('empty')}</p>
      ) : (
        <>
          <ul className='space-y-1'>
            {references.map((ref) => {
              // 业务模块在翻译文件里登记自己的 ownerType / field 名称；未登记时显示原始标识。
              const ownerKey = `ownerTypes.${ref.ownerType}`;
              const fieldKey = `fields.${ref.ownerType}.${ref.field}`;
              return (
                <li
                  key={`${ref.ownerType}:${ref.ownerId}:${ref.field}`}
                  className='flex items-center justify-between gap-2 text-sm'
                >
                  <span>
                    {t.has(ownerKey) ? t(ownerKey) : ref.ownerType}{' '}
                    <span className='text-muted-foreground'>#{ref.ownerId}</span>
                  </span>
                  <span className='text-muted-foreground text-xs'>
                    {t.has(fieldKey) ? t(fieldKey) : ref.field}
                  </span>
                </li>
              );
            })}
          </ul>
          <p className='text-muted-foreground text-xs'>{t('inUseHint')}</p>
        </>
      )}
    </section>
  );
}
