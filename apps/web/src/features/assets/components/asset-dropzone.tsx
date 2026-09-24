'use client';

import { useCallback, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Icons } from '@/components/icons';
import { cn, formatBytes } from '@/lib/utils';

interface AssetDropzoneProps {
  /** 当前选中的文件；受控，由调用方决定何时上传、何时清空。 */
  file: File | null;
  onFileChange: (file: File | null) => void;
  /** 原生 `<input accept>` 取值，如 `image/*`。仅用于过滤选择框，真正的校验在后端。 */
  accept?: string;
  disabled?: boolean;
  className?: string;
}

/** 单文件选择区：拖拽或浏览。资产上传面板与表单里的 AssetField 共用。 */
export function AssetDropzone({
  file,
  onFileChange,
  accept,
  disabled,
  className
}: AssetDropzoneProps) {
  const t = useTranslations('assets.upload');
  const inputRef = useRef<HTMLInputElement>(null);
  const [isDragging, setIsDragging] = useState(false);

  const clear = useCallback(() => {
    onFileChange(null);
    if (inputRef.current) inputRef.current.value = '';
  }, [onFileChange]);

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors',
        isDragging
          ? 'border-primary bg-primary/5'
          : 'border-muted-foreground/25 hover:border-muted-foreground/50',
        file && 'border-primary/50 bg-primary/5',
        disabled && 'pointer-events-none opacity-60',
        className
      )}
      onDragOver={(e) => {
        e.preventDefault();
        setIsDragging(true);
      }}
      onDragLeave={(e) => {
        e.preventDefault();
        setIsDragging(false);
      }}
      onDrop={(e) => {
        e.preventDefault();
        setIsDragging(false);
        const dropped = e.dataTransfer.files[0];
        if (dropped) onFileChange(dropped);
      }}
    >
      <input
        ref={inputRef}
        type='file'
        accept={accept}
        className='hidden'
        disabled={disabled}
        onChange={(e) => {
          const picked = e.target.files?.[0];
          if (picked) onFileChange(picked);
        }}
      />
      {file ? (
        <div className='flex flex-col items-center gap-3 text-center'>
          <Icons.page className='h-12 w-12 text-muted-foreground' />
          <div>
            <p className='font-medium'>{file.name}</p>
            <p className='text-muted-foreground text-sm'>
              {formatBytes(file.size)} &middot; {file.type || t('unknownType')}
            </p>
          </div>
          <Button type='button' variant='ghost' size='sm' onClick={clear}>
            <Icons.close className='mr-1 h-3 w-3' /> {t('remove')}
          </Button>
        </div>
      ) : (
        <div className='flex flex-col items-center gap-3 text-center'>
          <Icons.upload className='h-10 w-10 text-muted-foreground' />
          <div>
            <p className='font-medium'>{t('dropHere')}</p>
            <p className='text-muted-foreground text-sm'>{t('clickToBrowse')}</p>
          </div>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => inputRef.current?.click()}
          >
            {t('browseFiles')}
          </Button>
        </div>
      )}
    </div>
  );
}
