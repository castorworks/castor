'use client';
import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Icons } from '@/components/icons';
import { AlertModal } from '@/components/modal/alert-modal';
import { usePermission } from '@/hooks/use-permission';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { cn } from '@/lib/utils';
import { menuCatalogQuery } from '../api/queries';
import { deleteMenuMutation } from '../api/mutations';
import type { Menu } from '../api/types';
import { menuTree, menuTitle, type MenuNode } from '../tree';
import { MenuForm } from './menu-form';

export function MenuListing() {
  const t = useTranslations('menus');
  const tc = useTranslations('common');
  const locale = useLocale();
  const query = useQuery(menuCatalogQuery);
  const [editing, setEditing] = useState<Menu | null | undefined>();
  const [deleting, setDeleting] = useState<Menu | null>(null);
  const create = usePermission('/api/v1/admin/menus:POST');
  const update = usePermission('/api/v1/admin/menus/:id:PUT');
  const remove = usePermission('/api/v1/admin/menus/:id:DELETE');
  const mutation = useMutation({
    ...mergeMutationOptions(deleteMenuMutation, {
      onSuccess: () => {
        toast.success(t('deleted'));
        setDeleting(null);
      },
      onError: (error) => toast.error(error.message || t('failed'))
    })
  });
  const render = (nodes: MenuNode[], depth = 0): React.ReactNode =>
    nodes.map((node) => (
      <div key={node.id}>
        <div
          className={cn(
            'flex items-center gap-3 rounded-md border px-3 py-2',
            !node.isEnabled && 'opacity-60'
          )}
          style={{ marginLeft: depth * 16 }}
        >
          <Badge variant='outline'>{t(node.kind)}</Badge>
          <div className='min-w-0 flex-1'>
            <div className='truncate font-medium'>{menuTitle(node, locale)}</div>
            <div className='text-muted-foreground truncate text-xs'>{node.path || node.code}</div>
          </div>
          <span className='text-muted-foreground text-xs'>{node.sortOrder}</span>
          {!node.isEnabled ? <Badge variant='secondary'>{t('disabled')}</Badge> : null}
          {update ? (
            <Button
              size='sm'
              variant='ghost'
              aria-label={`${t('edit')} ${menuTitle(node, locale)}`}
              onClick={() => setEditing(node)}
            >
              <Icons.edit />
            </Button>
          ) : null}
          {remove ? (
            <Button
              size='sm'
              variant='ghost'
              disabled={node.children.length > 0}
              aria-label={`${tc('delete')} ${menuTitle(node, locale)}`}
              onClick={() => setDeleting(node)}
            >
              <Icons.trash />
            </Button>
          ) : null}
        </div>
        <div className='mt-2 space-y-2'>{render(node.children, depth + 1)}</div>
      </div>
    ));
  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-3'>
        <p className='text-muted-foreground text-sm'>{t('treeHelp')}</p>
        {create ? (
          <Button onClick={() => setEditing(null)}>
            <Icons.add />
            {t('create')}
          </Button>
        ) : null}
      </div>
      {query.isPending ? (
        <p>{tc('loading')}</p>
      ) : query.isError ? (
        <div role='alert'>
          {t('loadFailed')}{' '}
          <Button variant='outline' onClick={() => query.refetch()}>
            {t('retry')}
          </Button>
        </div>
      ) : query.data.menus.length ? (
        render(menuTree(query.data.menus))
      ) : (
        <p>{t('empty')}</p>
      )}
      {editing !== undefined && query.data ? (
        <MenuForm
          key={editing?.id ?? 'new'}
          menu={editing ?? undefined}
          catalog={query.data}
          onClose={() => setEditing(undefined)}
        />
      ) : null}
      <AlertModal
        isOpen={!!deleting}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (deleting) mutation.mutate(deleting.id);
        }}
        loading={mutation.isPending}
      />
    </div>
  );
}
