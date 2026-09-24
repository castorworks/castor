'use client';

import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Icons } from '@/components/icons';
import { AlertModal } from '@/components/modal/alert-modal';
import { usePermission } from '@/hooks/use-permission';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { cn } from '@/lib/utils';
import { deleteDepartmentMutation } from '../api/mutations';
import { departmentsQueryOptions } from '../api/queries';
import type { Department } from '../api/types';
import { departmentTree, type DepartmentNode } from '../tree';
import { DepartmentForm } from './department-form';

type Editing = { department?: Department; parentId?: number } | undefined;

export function DepartmentListing() {
  const t = useTranslations('departments');
  const tc = useTranslations('common');
  const query = useQuery(departmentsQueryOptions());
  const [editing, setEditing] = useState<Editing>();
  const [deleting, setDeleting] = useState<Department | null>(null);
  const [collapsed, setCollapsed] = useState<Set<number>>(new Set());
  const canCreate = usePermission('/api/v1/admin/departments:POST');
  const canUpdate = usePermission('/api/v1/admin/departments/:id:PUT');
  const canDelete = usePermission('/api/v1/admin/departments/:id:DELETE');
  const mutation = useMutation({
    ...mergeMutationOptions(deleteDepartmentMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleted'));
        setDeleting(null);
      },
      onError: (error) => toast.error(error.message || t('messages.deleteFailed'))
    })
  });

  const departments = query.data ?? [];
  // 全部数据范围时每个部门都在范围内，也只有这时能在根上新建部门。
  const canUseRoot = departments.every((d) => d.inScope);

  const toggle = (id: number) =>
    setCollapsed((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const render = (nodes: DepartmentNode[], depth = 0): React.ReactNode =>
    nodes.map((node) => (
      <div key={node.id} className='space-y-2'>
        <div
          className={cn(
            'flex items-center gap-3 rounded-md border px-3 py-2',
            (!node.isEnabled || !node.inScope) && 'opacity-60'
          )}
          style={{ marginLeft: depth * 20 }}
        >
          <Button
            size='icon'
            variant='ghost'
            className='size-6'
            disabled={node.children.length === 0}
            aria-label={collapsed.has(node.id) ? t('expand') : t('collapse')}
            onClick={() => toggle(node.id)}
          >
            {node.children.length === 0 ? (
              <Icons.circle className='size-2' />
            ) : collapsed.has(node.id) ? (
              <Icons.chevronRight />
            ) : (
              <Icons.chevronDown />
            )}
          </Button>
          <div className='min-w-0 flex-1'>
            <div className='truncate font-medium'>{node.name}</div>
            <code className='text-muted-foreground text-xs'>{node.code}</code>
          </div>
          <span className='text-muted-foreground text-xs'>
            {t('memberCount', { count: node.memberCount })}
          </span>
          {!node.isEnabled && <Badge variant='secondary'>{t('disabled')}</Badge>}
          {!node.inScope && <Badge variant='outline'>{t('outOfScope')}</Badge>}
          {canCreate && node.inScope && (
            <Button
              size='sm'
              variant='ghost'
              aria-label={t('addChildOf', { name: node.name })}
              onClick={() => setEditing({ parentId: node.id })}
            >
              <Icons.add />
            </Button>
          )}
          {canUpdate && node.inScope && (
            <Button
              size='sm'
              variant='ghost'
              aria-label={t('editNamed', { name: node.name })}
              onClick={() => setEditing({ department: node })}
            >
              <Icons.edit />
            </Button>
          )}
          {canDelete && node.inScope && (
            <Button
              size='sm'
              variant='ghost'
              disabled={node.children.length > 0 || node.memberCount > 0}
              aria-label={t('deleteNamed', { name: node.name })}
              title={
                node.children.length > 0 || node.memberCount > 0 ? t('deleteBlocked') : undefined
              }
              onClick={() => setDeleting(node)}
            >
              <Icons.trash />
            </Button>
          )}
        </div>
        {!collapsed.has(node.id) && render(node.children, depth + 1)}
      </div>
    ));

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-3'>
        <p className='text-muted-foreground text-sm'>{t('treeHelp')}</p>
        {canCreate && canUseRoot && (
          <Button onClick={() => setEditing({})}>
            <Icons.add /> {t('create')}
          </Button>
        )}
      </div>
      {query.isPending ? (
        <p>{tc('loading')}</p>
      ) : query.isError ? (
        <div role='alert' className='flex items-center gap-2'>
          {t('loadFailed')}
          <Button variant='outline' onClick={() => query.refetch()}>
            {t('retry')}
          </Button>
        </div>
      ) : departments.length ? (
        render(departmentTree(departments))
      ) : (
        <p className='text-muted-foreground text-sm'>{t('empty')}</p>
      )}
      {editing !== undefined && (
        <DepartmentForm
          key={editing.department?.id ?? `new-${editing.parentId ?? 'root'}`}
          department={editing.department}
          parentId={editing.parentId}
          departments={departments}
          canUseRoot={canUseRoot}
          onClose={() => setEditing(undefined)}
        />
      )}
      <AlertModal
        isOpen={!!deleting}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && mutation.mutate(deleting.id)}
        loading={mutation.isPending}
      />
    </div>
  );
}
