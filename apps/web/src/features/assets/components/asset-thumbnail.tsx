'use client';

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Icons } from '@/components/icons';
import { assetUrl } from '@/lib/asset-url';
import { cn } from '@/lib/utils';

interface AssetThumbnailProps {
  objectKey?: string;
  /** 只有图片才尝试预览，其余显示图标。 */
  isImage: boolean;
  icon?: React.ComponentType<{ className?: string }>;
  className?: string;
}

/**
 * 后台里的资产缩略图。走受 RBAC 保护的后台下载地址，因此私有图片也能预览；
 * 不用 next/image：它的优化器在服务端取图，不带用户 cookie，取不到受保护的文件。
 * 图片加载失败（无下载权限、文件已回收）时自动退回图标。
 */
export function AssetThumbnail({
  objectKey,
  isImage,
  icon: Icon = Icons.paperclip,
  className
}: AssetThumbnailProps) {
  return (
    <Avatar className={cn('h-10 w-10 shrink-0 rounded', className)}>
      {objectKey && isImage && (
        <AvatarImage
          src={assetUrl(objectKey, { admin: true })}
          alt=''
          className='rounded object-cover'
        />
      )}
      <AvatarFallback className='rounded'>
        <Icon className='h-5 w-5 text-muted-foreground' />
      </AvatarFallback>
    </Avatar>
  );
}
