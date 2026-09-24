'use client';

import { useAppForm } from '@/components/ui/tanstack-form';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import { useMutation } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { createDictItemMutation, updateDictItemMutation } from '../api/mutations';
import { toast } from 'sonner';
import {
  fromI18nText,
  getCreateDictItemSchema,
  toI18nText,
  type CreateDictItemFormValues
} from '../schemas/dictionary';
import { locales } from '@/i18n/config';
import { localizedText } from '@/lib/i18n-text';
import { TAG_COLORS, tagColorBg } from '@/lib/tag-color';
import { cn } from '@/lib/utils';
import type { DictItem } from '../api/types';
import { mergeMutationOptions } from '@/lib/mutation-utils';

// Radix Select 不接受空串取值，用哨兵表示"不设置"。
const NO_ICON = 'none';

interface DictItemFormSheetProps {
  typeCode: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existing?: DictItem | null;
}

function toIconKey(value: string | undefined): string {
  return !value || value === NO_ICON ? '' : value;
}

export function DictItemFormSheet({
  typeCode,
  open,
  onOpenChange,
  existing
}: DictItemFormSheetProps) {
  const t = useTranslations('dictionary');
  const tc = useTranslations('common');
  const locale = useLocale();
  const isEdit = !!existing;
  // zod schema 只持有 i18n key，翻译在组件里完成后注入。
  const validationMessages = {
    codeRequired: t('validation.codeRequired'),
    codeFormat: t('validation.codeFormat'),
    nameRequired: t('validation.nameRequired'),
    labelRequired: t('validation.labelRequired'),
    valueRequired: t('validation.valueRequired')
  };

  const createMutation = useMutation({
    ...mergeMutationOptions(createDictItemMutation, {
      onSuccess: () => {
        toast.success(t('messages.itemCreated'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.itemCreateFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateDictItemMutation, {
      onSuccess: () => {
        toast.success(t('messages.itemUpdated'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.itemUpdateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: {
      ...fromI18nText(existing?.label),
      value: existing?.value ?? '',
      description: existing?.description ?? '',
      color: existing?.color ?? '',
      icon: existing?.icon || NO_ICON,
      isDefault: existing?.isDefault ?? false,
      sortOrder: existing?.sortOrder ?? 0
    } as CreateDictItemFormValues,
    validators: {
      onSubmit: getCreateDictItemSchema(validationMessages)
    },
    onSubmit: async ({ value }) => {
      const v = value as CreateDictItemFormValues;
      if (isEdit && existing) {
        await updateMutation.mutateAsync({
          id: existing.id,
          values: {
            label: toI18nText(v),
            // 系统项的取值被代码引用，后端会拒绝修改；这里干脆不提交。
            value: existing.isSystem ? undefined : v.value,
            description: v.description || undefined,
            // 编辑时空串表示清除，不能省略成 undefined
            color: v.color ?? '',
            icon: toIconKey(v.icon),
            isDefault: v.isDefault ?? false,
            sortOrder: v.sortOrder ?? 0
          }
        });
      } else {
        await createMutation.mutateAsync({
          typeCode,
          label: toI18nText(v),
          value: v.value,
          description: v.description || undefined,
          color: v.color || undefined,
          icon: toIconKey(v.icon) || undefined,
          isDefault: v.isDefault ?? false,
          sortOrder: v.sortOrder ?? 0
        });
      }
    }
  });

  return (
    <Sheet key={existing?.id ?? 'create'} open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEdit ? t('item.editTitle') : t('item.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEdit
              ? t('item.editExisting', { label: localizedText(existing?.label, locale) })
              : t('item.newDescription', { typeCode })}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='dict-item-form-sheet' className='space-y-4'>
              <div className='grid gap-3 sm:grid-cols-2'>
                {locales.map((lang) => (
                  <form.TextField
                    key={lang}
                    name={lang}
                    label={t(`fields.label_${lang}`)}
                    required
                    placeholder={t('fields.itemLabelPlaceholder')}
                  />
                ))}
              </div>
              <form.TextField
                name='value'
                label={t('item.value')}
                required
                placeholder={t('fields.itemValuePlaceholder')}
                disabled={existing?.isSystem}
                description={existing?.isSystem ? t('item.systemValueLocked') : undefined}
              />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('fields.describeItem')}
              />
              <form.AppField name='color'>
                {(field) => (
                  <field.FieldSet>
                    <field.Field>
                      <field.FieldLabel>{t('fields.color')}</field.FieldLabel>
                      <div role='radiogroup' className='flex flex-wrap items-center gap-2'>
                        {['', ...TAG_COLORS].map((color) => {
                          const selected = (field.state.value ?? '') === color;
                          const name = color ? t(`colors.${color}`) : t('fields.none');
                          return (
                            <button
                              key={color || 'none'}
                              type='button'
                              role='radio'
                              aria-checked={selected}
                              aria-label={name}
                              title={name}
                              onClick={() => field.handleChange(color)}
                              className={cn(
                                'flex size-7 items-center justify-center rounded-full border transition-shadow',
                                'focus-visible:ring-ring/50 outline-none focus-visible:ring-[3px]',
                                tagColorBg(color),
                                selected && 'ring-ring ring-2 ring-offset-2 ring-offset-background'
                              )}
                            >
                              {!color && <Icons.close className='text-muted-foreground size-3.5' />}
                            </button>
                          );
                        })}
                      </div>
                    </field.Field>
                  </field.FieldSet>
                )}
              </form.AppField>
              <form.SelectField
                name='icon'
                label={t('fields.icon')}
                options={[
                  { value: NO_ICON, label: t('fields.none') },
                  ...Object.keys(Icons).map((value) => ({ value, label: value }))
                ]}
              />
              <form.SwitchField
                name='isDefault'
                label={t('fields.default')}
                description={t('item.defaultOption')}
              />
              <form.TextField
                name='sortOrder'
                label={t('fields.sortOrder')}
                type='number'
                placeholder='0'
              />
            </form.Form>
          </form.AppForm>
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button
            type='submit'
            form='dict-item-form-sheet'
            isLoading={createMutation.isPending || updateMutation.isPending}
          >
            <Icons.check /> {isEdit ? t('item.updateItem') : t('item.createItem')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
