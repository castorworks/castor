'use client';

import { Icons } from '@/components/icons';
import { useDict } from '@/hooks/use-dict';
import { cn } from '@/lib/utils';
import type { AssetAttachment } from '../api/types';

interface AssetAttachmentListProps {
  items: AssetAttachment[];
  /** 下载地址由拥有这些附件的业务模块给出（它自己的、带鉴权的下载路由）；不给则只展示不可点。 */
  hrefFor?: (item: AssetAttachment) => string | undefined;
  /** 每一项右侧的操作（如表单里的“移除”）。 */
  renderAction?: (item: AssetAttachment) => React.ReactNode;
  className?: string;
}

/** 业务记录上的附件列表：文件名 + 大小，图标随资产分类。展示与表单共用。 */
export function AssetAttachmentList({
  items,
  hrefFor,
  renderAction,
  className
}: AssetAttachmentListProps) {
  const dict = useDict();
  if (items.length === 0) return null;

  return (
    <ul className={cn('flex flex-col gap-1', className)}>
      {items.map((item) => {
        const iconKey = dict.icon('asset_category', item.category) as
          | keyof typeof Icons
          | undefined;
        const Icon = iconKey ? (Icons[iconKey] ?? Icons.paperclip) : Icons.paperclip;
        const href = hrefFor?.(item);
        const label = (
          <>
            <Icon className='h-4 w-4 shrink-0 text-muted-foreground' />
            <span className='truncate'>{item.name}</span>
            <span className='text-muted-foreground shrink-0 text-xs'>{item.sizeFormatted}</span>
          </>
        );
        return (
          <li key={item.objectKey} className='flex items-center gap-2 text-sm'>
            {href ? (
              <a
                href={href}
                download
                className='flex min-w-0 flex-1 items-center gap-2 underline-offset-2 hover:underline'
              >
                {label}
              </a>
            ) : (
              <span className='flex min-w-0 flex-1 items-center gap-2'>{label}</span>
            )}
            {renderAction?.(item)}
          </li>
        );
      })}
    </ul>
  );
}
