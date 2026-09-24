'use client';

import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Icons } from '@/components/icons';
import { AlertModal } from '@/components/modal/alert-modal';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { EmptyState } from '@/components/ui/empty-state';
import { usePermission } from '@/hooks/use-permission';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { deleteConstraintMutation } from '../api/mutations';
import { constraintsQueryOptions, rolesQueryOptions } from '../api/queries';
import type { SeparationConstraint } from '../api/types';
import { ConstraintFormSheet } from './constraint-form-sheet';

export function ConstraintListing() {
  const t = useTranslations('authorization.constraints');
  const { data: constraints = [], isLoading } = useQuery(constraintsQueryOptions());
  const { data: roles = [] } = useQuery(rolesQueryOptions());
  const roleById = new Map(roles.map((role) => [role.id, role]));

  if (isLoading) {
    return <Icons.spinner className='mx-auto size-6 animate-spin' />;
  }
  if (!constraints.length) {
    return <EmptyState icon={<Icons.shield />} title={t('empty')} description={t('emptyHelp')} />;
  }
  return (
    <div className='grid gap-4 lg:grid-cols-2'>
      {constraints.map((constraint) => (
        <ConstraintCard
          key={constraint.id}
          constraint={constraint}
          roleNames={constraint.roleIds.map((id) => roleById.get(id)?.name ?? String(id))}
        />
      ))}
    </div>
  );
}

function ConstraintCard({
  constraint,
  roleNames
}: {
  constraint: SeparationConstraint;
  roleNames: string[];
}) {
  const t = useTranslations('authorization.constraints');
  const tc = useTranslations('common');
  const { ts } = useSeedTranslation();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const canEdit = usePermission('/api/v1/admin/authorization/constraints/:id:PUT');
  const canDelete = usePermission('/api/v1/admin/authorization/constraints/:id:DELETE');
  const deletion = useMutation({
    ...mergeMutationOptions(deleteConstraintMutation, {
      onSuccess: () => toast.success(t('messages.deleted')),
      onError: () => toast.error(t('messages.deleteFailed'))
    })
  });

  return (
    <>
      <Card className={!constraint.isEnabled ? 'opacity-60' : ''}>
        <CardHeader>
          <div className='flex items-start justify-between gap-2'>
            <div>
              <CardTitle>{constraint.name}</CardTitle>
              <CardDescription className='font-mono'>{constraint.code}</CardDescription>
            </div>
            <div className='flex gap-1'>
              <Badge variant={constraint.type === 'SSD' ? 'default' : 'secondary'}>
                {constraint.type}
              </Badge>
              <Badge variant={constraint.isEnabled ? 'outline' : 'destructive'}>
                {constraint.isEnabled ? tc('enabled') : tc('disabled')}
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent className='space-y-4'>
          {constraint.description && (
            <p className='text-sm text-muted-foreground'>{constraint.description}</p>
          )}
          <div className='flex flex-wrap gap-2'>
            {roleNames.map((name, index) => (
              <Badge key={`${name}-${index}`} variant='outline'>
                {ts(name)}
              </Badge>
            ))}
          </div>
          <p className='text-sm text-muted-foreground'>
            {t('ruleSummary', { count: constraint.cardinality })}
          </p>
          <div className='flex justify-end gap-2'>
            {canEdit && (
              <Button size='sm' variant='outline' onClick={() => setEditOpen(true)}>
                <Icons.edit /> {tc('edit')}
              </Button>
            )}
            {canDelete && (
              <Button size='sm' variant='destructive' onClick={() => setDeleteOpen(true)}>
                <Icons.trash /> {tc('delete')}
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
      <ConstraintFormSheet constraint={constraint} open={editOpen} onOpenChange={setEditOpen} />
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deletion.mutate(constraint.id)}
        loading={deletion.isPending}
      />
    </>
  );
}
