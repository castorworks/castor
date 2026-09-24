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
import { AssetPickerDialog } from '@/features/assets/components/asset-picker-dialog';
import { AssetThumbnail } from '@/features/assets/components/asset-thumbnail';
import { createAssetMutation } from '@/features/assets/api/mutations';
import type { AssetCategory } from '@/features/assets/api/types';
import { usePermission } from '@/hooks/use-permission';
import { assetUrl } from '@/lib/asset-url';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface AssetFieldProps {
  label: string;
  description?: string;
  required?: boolean;
  /** 限定分类：同时约束资产库选择器与图片预览，如头像、封面传 `IMAGE`。 */
  category?: AssetCategory;
  /** 原生 `<input accept>`，仅过滤文件选择框；真正的类型校验在后端。 */
  accept?: string;
  /** 新上传的文件是否公开。要经免认证地址展示的字段（头像、封面）传 true。 */
  isPublic?: boolean;
  disabled?: boolean;
}

/**
 * 业务表单里的文件字段。字段值是资产的 `objectKey`（字符串）：
 * 可以现场上传（作为业务附件，随引用回收），也可以从资产库挑已有文件。
 *
 * 前端只负责拿到 objectKey；引用登记（Attach / Replace）由后端业务服务在保存记录时完成，
 * 见 apps/api/AGENTS.md 的 "Files in business modules (assets)"。
 *
 * 用法：`<FormAssetField name='cover' label=… category='IMAGE' isPublic />`；
 * 需要类型安全的字段名时用 `typedField<FormValues>()(FormAssetField)`
 * （参考 features/users/components/user-form-sheet.tsx）。
 */
export function AssetField({
  label,
  description,
  required,
  category,
  accept,
  isPublic = false,
  disabled
}: AssetFieldProps) {
  const t = useTranslations('assets.field');
  const field = useFieldContext();
  const value = (useStore(field.store, (s) => s.value) as string | undefined) ?? '';
  const inputRef = useRef<HTMLInputElement>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const canUpload = usePermission('/api/v1/admin/assets:POST');
  const canBrowse = usePermission('/api/v1/admin/assets:GET');

  const upload = useMutation({
    ...mergeMutationOptions(createAssetMutation, {
      onSuccess: (asset) => {
        field.handleChange(asset.objectKey);
        field.handleBlur();
      },
      onError: (error) => toast.error(error.message || t('uploadFailed'))
    })
  });

  const handleFile = (file: File | undefined) => {
    if (!file) return;
    const formData = new FormData();
    formData.append('file', file);
    // 表单里上传的文件属于这条业务记录，不进资产库：被替换或记录删除后随引用回收。
    formData.append('scope', 'ATTACHMENT');
    if (isPublic) formData.append('isPublic', 'true');
    upload.mutate(formData);
    if (inputRef.current) inputRef.current.value = '';
  };

  const busy = disabled || upload.isPending;

  return (
    <FormFieldSet>
      <FormField>
        <FieldLabel htmlFor={field.name}>
          {label}
          {required && ' *'}
        </FieldLabel>

        <div className='flex items-center gap-3 rounded-lg border p-3'>
          <AssetThumbnail objectKey={value} isImage={category === 'IMAGE'} className='h-12 w-12' />

          <div className='min-w-0 flex-1'>
            {value ? (
              <a
                href={assetUrl(value, { admin: true })}
                download
                className='block truncate text-sm underline-offset-2 hover:underline'
              >
                {value}
              </a>
            ) : (
              <p className='text-muted-foreground text-sm'>{t('empty')}</p>
            )}
          </div>

          <div className='flex shrink-0 gap-1'>
            <input
              ref={inputRef}
              id={field.name}
              type='file'
              accept={accept}
              className='hidden'
              // 标签的 htmlFor 指向这个 input：没有上传权限或正忙时必须禁用，否则点标签仍能选文件。
              disabled={busy || !canUpload}
              onChange={(e) => handleFile(e.target.files?.[0])}
            />
            {canUpload && (
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={busy}
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
                disabled={busy}
                onClick={() => setPickerOpen(true)}
              >
                <Icons.workspace className='mr-1 h-3.5 w-3.5' /> {t('browse')}
              </Button>
            )}
            {value && !required && (
              <Button
                type='button'
                variant='ghost'
                size='sm'
                disabled={busy}
                aria-label={t('clear')}
                onClick={() => {
                  field.handleChange('');
                  field.handleBlur();
                }}
              >
                <Icons.close className='h-3.5 w-3.5' />
              </Button>
            )}
          </div>
        </div>

        {description && <FieldDescription>{description}</FieldDescription>}
      </FormField>
      <FormFieldError />

      {canBrowse && (
        <AssetPickerDialog
          open={pickerOpen}
          onOpenChange={setPickerOpen}
          category={category}
          selectedKey={value}
          onSelect={(asset) => {
            field.handleChange(asset.objectKey);
            field.handleBlur();
          }}
        />
      )}
    </FormFieldSet>
  );
}

export const FormAssetField = createFormField(AssetField);
