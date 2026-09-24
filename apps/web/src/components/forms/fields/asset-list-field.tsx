'use client';

import { useRef, useState } from 'react';
import { useStore } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { FieldDescription, FieldLabel } from '@/components/ui/field';
import {
  useFieldContext,
  FormFieldSet,
  FormField,
  FormFieldError,
  createFormField
} from '@/components/ui/form-context';
import { Icons } from '@/components/icons';
import { AssetAttachmentList } from '@/features/assets/components/asset-attachment-list';
import { AssetPickerDialog } from '@/features/assets/components/asset-picker-dialog';
import { uploadAttachment } from '@/features/assets/api/service';
import type { AssetAttachment } from '@/features/assets/api/types';
import { usePermission } from '@/hooks/use-permission';

interface AssetListFieldProps {
  label: string;
  description?: string;
  /**
   * 拥有这个字段的业务模块自己的上传路由（不含 `/api` 前缀），如
   * `/v1/admin/notifications/attachments`。由它的权限决定谁能上传，而不是资产库的权限。
   */
  uploadPath: string;
  /** 上传路由对应的权限标识，用于显隐上传按钮。 */
  uploadPermission: string;
  /** 最多几个文件。 */
  max?: number;
  /** 原生 `<input accept>`，仅过滤文件选择框；真正的类型校验在后端。 */
  accept?: string;
  disabled?: boolean;
  /** 已保存到记录上的附件的下载地址（业务模块自己的鉴权路由）；刚上传、尚未保存的返回 undefined。 */
  hrefFor?: (item: AssetAttachment) => string | undefined;
}

/**
 * 业务表单里的多文件字段。字段值是 `AssetAttachment[]`；提交时把每一项的
 * `{ objectKey, name }` 交给后端，由业务服务用 `AssetReferencer.Sync` 登记引用。
 * 单文件字段用 `AssetField`。
 */
export function AssetListField({
  label,
  description,
  uploadPath,
  uploadPermission,
  max = 10,
  accept,
  disabled,
  hrefFor
}: AssetListFieldProps) {
  const t = useTranslations('assets.field');
  const field = useFieldContext();
  const value = (useStore(field.store, (s) => s.value) as AssetAttachment[] | undefined) ?? [];
  const inputRef = useRef<HTMLInputElement>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const canUpload = usePermission(uploadPermission);
  const canBrowse = usePermission('/api/v1/admin/assets:GET');

  const add = (item: AssetAttachment) => {
    // 同一份内容只会有一个对象键（后端按哈希去重），重复添加没有意义。
    const current = (field.store.state.value as AssetAttachment[] | undefined) ?? [];
    if (current.some((existing) => existing.objectKey === item.objectKey)) return;
    field.handleChange([...current, item]);
    field.handleBlur();
  };

  const upload = useMutation({
    mutationFn: (file: File) => uploadAttachment(uploadPath, file),
    onSuccess: add,
    onError: (error) => toast.error(error.message || t('uploadFailed'))
  });

  const full = value.length >= max;
  const busy = disabled || upload.isPending;

  return (
    <FormFieldSet>
      <FormField>
        <FieldLabel htmlFor={field.name}>{label}</FieldLabel>

        <div className='space-y-3 rounded-lg border p-3'>
          {value.length === 0 ? (
            <p className='text-muted-foreground text-sm'>{t('emptyList')}</p>
          ) : (
            <AssetAttachmentList
              items={value}
              hrefFor={hrefFor}
              renderAction={(item) => (
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  className='h-6 w-6 shrink-0 p-0'
                  disabled={busy}
                  aria-label={t('remove', { name: item.name })}
                  onClick={() => {
                    field.handleChange(value.filter((v) => v.objectKey !== item.objectKey));
                    field.handleBlur();
                  }}
                >
                  <Icons.close className='h-3.5 w-3.5' />
                </Button>
              )}
            />
          )}

          <div className='flex items-center gap-2'>
            <input
              ref={inputRef}
              id={field.name}
              type='file'
              accept={accept}
              className='hidden'
              // 标签的 htmlFor 指向这个 input：不能上传时必须禁用，否则点标签仍能选文件。
              disabled={busy || full || !canUpload}
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) upload.mutate(file);
                e.target.value = '';
              }}
            />
            {canUpload && (
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={busy || full}
                isLoading={upload.isPending}
                onClick={() => inputRef.current?.click()}
              >
                <Icons.upload className='mr-1 h-3.5 w-3.5' /> {t('upload')}
              </Button>
            )}
            {canBrowse && (
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={busy || full}
                onClick={() => setPickerOpen(true)}
              >
                <Icons.workspace className='mr-1 h-3.5 w-3.5' /> {t('browse')}
              </Button>
            )}
            <span className='text-muted-foreground ml-auto text-xs'>
              {t('count', { count: value.length, max })}
            </span>
          </div>
        </div>

        {description && <FieldDescription>{description}</FieldDescription>}
      </FormField>
      <FormFieldError />

      {canBrowse && (
        <AssetPickerDialog
          open={pickerOpen}
          onOpenChange={setPickerOpen}
          onSelect={(asset) =>
            add({
              objectKey: asset.objectKey,
              name: asset.filename,
              extension: asset.extension,
              mimeType: asset.mimeType,
              size: asset.size,
              sizeFormatted: asset.sizeFormatted,
              category: asset.category
            })
          }
        />
      )}
    </FormFieldSet>
  );
}

export const FormAssetListField = createFormField(AssetListField);
