'use client';
import { useMutation } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { useAppForm } from '@/components/ui/tanstack-form';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { saveMenuMutation } from '../api/mutations';
import type { Menu, MenuCatalog, MenuKind } from '../api/types';
import { menuTitle, permissionKey } from '../tree';
import { menuSchema } from '../schemas/menu';

export function MenuForm({
  menu,
  catalog,
  onClose
}: {
  menu?: Menu;
  catalog: MenuCatalog;
  onClose: () => void;
}) {
  const t = useTranslations('menus');
  const tc = useTranslations('common');
  const locale = useLocale();
  const { ts } = useSeedTranslation();
  const mutation = useMutation({
    ...mergeMutationOptions(saveMenuMutation, {
      onSuccess: () => {
        toast.success(t('saved'));
        onClose();
      },
      onError: (error) => toast.error(error.message || t('failed'))
    })
  });
  const form = useAppForm({
    defaultValues: {
      code: menu?.code ?? '',
      kind: menu?.kind ?? ('page' as MenuKind),
      parentId: menu?.parentId?.toString() ?? 'root',
      en: menu?.titles.en ?? '',
      zh: menu?.titles.zh ?? '',
      ja: menu?.titles.ja ?? '',
      ko: menu?.titles.ko ?? '',
      path: menu?.path ?? '',
      icon: menu?.icon ?? 'page',
      sortOrder: menu?.sortOrder ?? 0,
      isEnabled: menu?.isEnabled ?? true,
      accessMode: menu?.accessMode ?? ('permission' as 'authenticated' | 'permission'),
      permissions: menu?.permissions ?? []
    },
    validators: { onSubmit: menuSchema(t) },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        id: menu?.id,
        data: {
          code: value.code,
          kind: value.kind,
          parentId: value.parentId === 'root' ? null : Number(value.parentId),
          titles: { en: value.en, zh: value.zh, ja: value.ja, ko: value.ko },
          path: value.kind === 'page' ? value.path : '',
          icon: value.icon,
          sortOrder: value.sortOrder,
          isEnabled: value.isEnabled,
          accessMode:
            value.kind === 'directory'
              ? 'authenticated'
              : value.kind === 'action'
                ? 'permission'
                : value.accessMode,
          permissions:
            value.kind === 'directory' ||
            (value.kind === 'page' && value.accessMode === 'authenticated')
              ? []
              : value.permissions
        }
      });
    }
  });
  return (
    <Sheet
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent className='flex w-full flex-col sm:max-w-2xl'>
        <SheetHeader>
          <SheetTitle>{menu ? t('edit') : t('create')}</SheetTitle>
          <SheetDescription>{t('formDescription')}</SheetDescription>
        </SheetHeader>
        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='menu-form' className='space-y-4'>
              <form.TextField name='code' label={t('code')} required />
              <form.SelectField
                name='kind'
                label={t('kind')}
                options={(['directory', 'page', 'action'] as const).map((value) => ({
                  value,
                  label: t(value)
                }))}
              />
              <form.Subscribe selector={(state) => state.values.kind}>
                {(kind) => (
                  <form.SelectField
                    name='parentId'
                    label={t('parent')}
                    options={[
                      ...(kind === 'action' ? [] : [{ value: 'root', label: t('root') }]),
                      ...catalog.menus
                        .filter(
                          (item) =>
                            item.id !== menu?.id &&
                            item.kind === (kind === 'action' ? 'page' : 'directory')
                        )
                        .map((item) => ({ value: String(item.id), label: menuTitle(item, locale) }))
                    ]}
                  />
                )}
              </form.Subscribe>
              <div className='grid gap-3 sm:grid-cols-2'>
                {(['en', 'zh', 'ja', 'ko'] as const).map((lang) => (
                  <form.TextField key={lang} name={lang} label={t(`title_${lang}`)} required />
                ))}
              </div>
              <form.Subscribe selector={(state) => state.values.kind}>
                {(kind) =>
                  kind === 'page' ? (
                    <>
                      <form.TextField
                        name='path'
                        label={t('path')}
                        description={t('pathHelp')}
                        required
                      />
                      <form.SelectField
                        name='accessMode'
                        label={t('access')}
                        options={[
                          { value: 'permission', label: t('permissionAccess') },
                          { value: 'authenticated', label: t('authenticatedAccess') }
                        ]}
                      />
                    </>
                  ) : null
                }
              </form.Subscribe>
              <form.SelectField
                name='icon'
                label={t('icon')}
                options={Object.keys(Icons).map((value) => ({ value, label: value }))}
              />
              <form.TextField name='sortOrder' label={t('sortOrder')} type='number' />
              <form.SwitchField name='isEnabled' label={t('enabled')} />
              <form.Subscribe
                selector={(state) =>
                  [state.values.kind, state.values.accessMode, state.values.permissions] as const
                }
              >
                {([kind, accessMode, permissions]) =>
                  kind !== 'directory' && (kind === 'action' || accessMode === 'permission') ? (
                    <fieldset className='space-y-2'>
                      <legend className='text-sm font-medium'>{t('permissions')}</legend>
                      <p className='text-muted-foreground text-xs'>{t('permissionHelp')}</p>
                      <div className='max-h-72 space-y-3 overflow-auto rounded-md border p-3'>
                        {catalog.resources.map((resource) => (
                          <div key={resource.id}>
                            <span className='text-sm'>{ts(resource.name)}</span>
                            <div className='mt-1 flex flex-wrap gap-3'>
                              {resource.actions.map((action) => {
                                const ref = { resourceId: resource.id, action };
                                const checked = permissions.some(
                                  (p) => permissionKey(p) === permissionKey(ref)
                                );
                                return (
                                  <label key={action} className='flex items-center gap-1 text-xs'>
                                    <Checkbox
                                      checked={checked}
                                      onCheckedChange={() =>
                                        form.setFieldValue(
                                          'permissions',
                                          checked
                                            ? permissions.filter(
                                                (p) => permissionKey(p) !== permissionKey(ref)
                                              )
                                            : [...permissions, ref]
                                        )
                                      }
                                    />
                                    {action}
                                  </label>
                                );
                              })}
                            </div>
                          </div>
                        ))}
                      </div>
                      {!permissions.length ? (
                        <p className='text-destructive text-sm'>{t('permissionsRequired')}</p>
                      ) : null}
                    </fieldset>
                  ) : null
                }
              </form.Subscribe>
            </form.Form>
          </form.AppForm>
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={onClose}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='menu-form' isLoading={mutation.isPending}>
            {tc('save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
