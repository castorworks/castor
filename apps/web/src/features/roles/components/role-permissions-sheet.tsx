'use client';
import { useMemo, useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetDescription,
  SheetTitle,
  SheetFooter
} from '@/components/ui/sheet';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { usePermission } from '@/hooks/use-permission';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { menuTitle, nodePermissions, permissionKey, type MenuNode } from '@/features/menus/tree';
import {
  permissionTree,
  togglePermissions,
  selectionGrants
} from '@/features/menus/permission-tree';
import { rolePermissionsQueryOptions } from '../api/queries';
import { setRolePermissionsMutation } from '../api/mutations';
import type { Role, RolePermissionCatalog } from '../api/types';
import type { MenuPermission } from '@/features/menus/api/types';

export function RolePermissionsSheet({
  role,
  open,
  onOpenChange
}: {
  role: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations('roles.permissions');
  const tm = useTranslations('menus');
  const { ts } = useSeedTranslation();
  const query = useQuery({ ...rolePermissionsQueryOptions(role.code), enabled: open });
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex w-full flex-col sm:max-w-2xl'>
        <SheetHeader>
          <SheetTitle>
            {ts(role.name)} - {t('titleSuffix')}
          </SheetTitle>
          <SheetDescription>{tm('grantHelp')}</SheetDescription>
        </SheetHeader>
        {query.isPending ? (
          <div className='flex justify-center p-8'>
            <Icons.spinner className='animate-spin' />
          </div>
        ) : query.isError ? (
          <div role='alert'>
            {tm('loadFailed')}
            <Button onClick={() => query.refetch()}>{tm('retry')}</Button>
          </div>
        ) : open ? (
          <PermissionEditor
            key={role.code}
            role={role}
            catalog={query.data}
            onClose={() => onOpenChange(false)}
          />
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
function PermissionEditor({
  role,
  catalog,
  onClose
}: {
  role: Role;
  catalog: RolePermissionCatalog;
  onClose: () => void;
}) {
  const t = useTranslations('menus');
  const tp = useTranslations('roles.permissions');
  const tc = useTranslations('common');
  const messages = useTranslations('roles.messages');
  const locale = useLocale();
  const { ts } = useSeedTranslation();
  const canEdit = usePermission('/api/v1/admin/roles/:role/permissions:PUT');
  const [selection, setSelection] = useState(
    () =>
      new Set(
        catalog.grants.flatMap((grant) =>
          grant.actions.map((action) => permissionKey({ resourceId: grant.resourceId, action }))
        )
      )
  );
  const tree = useMemo(() => permissionTree(catalog, t('unmapped')), [catalog, t]);
  const menuTreeNodes = useMemo(() => tree.filter((node) => node.id !== -2147483647), [tree]);
  const advancedTreeNodes = useMemo(() => tree.filter((node) => node.id === -2147483647), [tree]);
  const allPermissions = useMemo(() => {
    const permissions = new Map<string, MenuPermission>();
    for (const node of tree) {
      for (const permission of nodePermissions(node)) {
        permissions.set(permissionKey(permission), permission);
      }
    }
    return [...permissions.values()];
  }, [tree]);
  const selectedCount = allPermissions.filter((permission) =>
    selection.has(permissionKey(permission))
  ).length;
  const mutation = useMutation({
    ...mergeMutationOptions(setRolePermissionsMutation, {
      onSuccess: () => {
        toast.success(messages('permissionsUpdated'));
        onClose();
      },
      onError: () => toast.error(messages('permissionsUpdateFailed'))
    })
  });
  const render = (nodes: MenuNode[], depth = 0): React.ReactNode =>
    nodes.map((node) => {
      const refs = nodePermissions(node);
      const count = refs.filter((p) => selection.has(permissionKey(p))).length;
      const checked =
        count > 0 && count < refs.length
          ? 'indeterminate'
          : refs.length > 0 && count === refs.length;
      const title = ts(menuTitle(node, locale));
      return (
        <div key={node.id} className='space-y-2'>
          <div className='rounded-md border p-3' style={{ marginLeft: depth * 16 }}>
            <label className='flex items-center gap-3'>
              <Checkbox
                aria-label={title}
                checked={checked}
                disabled={!canEdit || !refs.length || mutation.isPending}
                onCheckedChange={() => setSelection((current) => togglePermissions(current, refs))}
              />
              <span className='min-w-0 flex-1 truncate font-medium'>{title}</span>
              <Badge variant='outline'>{t(node.kind)}</Badge>
              {refs.length > 0 && (
                <span className='text-muted-foreground text-xs'>
                  {tp('operationSummary', {
                    selected: refs.filter((permission) => selection.has(permissionKey(permission)))
                      .length,
                    total: refs.length
                  })}
                </span>
              )}
              {!node.isEnabled ? <Badge variant='secondary'>{t('disabled')}</Badge> : null}
            </label>
            {node.kind === 'page' && refs.length > 0 ? (
              <details className='mt-2 ml-7'>
                <summary className='text-muted-foreground cursor-pointer text-xs'>
                  {tp('showOperations')}
                </summary>
                <div className='mt-2 flex flex-wrap gap-3'>
                  {refs.map((ref) => (
                    <label key={permissionKey(ref)} className='flex items-center gap-1 text-xs'>
                      <Checkbox
                        checked={selection.has(permissionKey(ref))}
                        disabled={!canEdit || mutation.isPending}
                        onCheckedChange={() =>
                          setSelection((current) => togglePermissions(current, [ref]))
                        }
                      />
                      {catalog.resources.find((resource) => resource.id === ref.resourceId)?.code ??
                        tp('unknownResource')}{' '}
                      · {ref.action}
                    </label>
                  ))}
                </div>
              </details>
            ) : node.kind === 'page' ? (
              <p className='text-muted-foreground mt-1 pl-7 text-xs'>{t('authenticatedAccess')}</p>
            ) : node.kind === 'action' ? (
              <div className='mt-2 flex flex-wrap gap-3 pl-7'>
                {refs.map((ref) => (
                  <label key={permissionKey(ref)} className='flex items-center gap-1 text-xs'>
                    <Checkbox
                      checked={selection.has(permissionKey(ref))}
                      disabled={!canEdit || mutation.isPending}
                      onCheckedChange={() =>
                        setSelection((current) => togglePermissions(current, [ref]))
                      }
                    />
                    {catalog.resources.find((resource) => resource.id === ref.resourceId)?.code ??
                      tp('unknownResource')}{' '}
                    · {ref.action}
                  </label>
                ))}
              </div>
            ) : null}
          </div>
          {node.kind !== 'page' ? render(node.children, depth + 1) : null}
        </div>
      );
    });
  return (
    <>
      <div className='text-muted-foreground mb-3 flex items-center justify-between text-xs'>
        <span>{tp('directPermissionsHelp')}</span>
        <Badge variant='secondary'>
          {tp('selectedSummary', { selected: selectedCount, total: allPermissions.length })}
        </Badge>
      </div>
      <Tabs defaultValue='menus' className='min-h-0 flex-1'>
        <TabsList className='grid w-full grid-cols-2'>
          <TabsTrigger value='menus'>{tp('menuTab')}</TabsTrigger>
          <TabsTrigger value='advanced'>{tp('advancedTab')}</TabsTrigger>
        </TabsList>
        <TabsContent value='menus' className='min-h-0 space-y-2 overflow-auto px-1 pt-3'>
          <p className='text-muted-foreground text-xs'>{tp('menuDescription')}</p>
          {menuTreeNodes.length > 0 ? (
            render(menuTreeNodes)
          ) : (
            <p className='text-muted-foreground py-8 text-center text-sm'>{tp('menuEmpty')}</p>
          )}
        </TabsContent>
        <TabsContent value='advanced' className='min-h-0 space-y-2 overflow-auto px-1 pt-3'>
          <p className='text-muted-foreground text-xs'>{tp('advancedDescription')}</p>
          {advancedTreeNodes.length > 0 ? (
            render(advancedTreeNodes)
          ) : (
            <p className='text-muted-foreground py-8 text-center text-sm'>{tp('advancedEmpty')}</p>
          )}
        </TabsContent>
      </Tabs>
      <SheetFooter>
        <Button variant='outline' onClick={onClose}>
          {tc('cancel')}
        </Button>
        {canEdit ? (
          <Button
            isLoading={mutation.isPending}
            onClick={() => mutation.mutate({ code: role.code, grants: selectionGrants(selection) })}
          >
            {tc('save')}
          </Button>
        ) : null}
      </SheetFooter>
    </>
  );
}
