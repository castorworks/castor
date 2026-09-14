'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { usePermission } from '@/hooks/use-permission';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { setRoleHierarchyMutation } from '../api/mutations';
import { roleHierarchyQueryOptions, rolesQueryOptions } from '../api/queries';
import type { Role } from '../api/types';

interface RoleHierarchySheetProps {
  role: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const EMPTY_ROLES: Role[] = [];

export function RoleHierarchySheet({ role, open, onOpenChange }: RoleHierarchySheetProps) {
  const t = useTranslations('roles.hierarchy');
  const tm = useTranslations('roles.messages');
  const tc = useTranslations('common');
  const { ts } = useSeedTranslation();
  const [selected, setSelected] = useState<string[]>([]);
  const canEdit = usePermission('/api/v1/admin/roles/:role/hierarchy:PUT');
  const { data: allRoles = EMPTY_ROLES, isLoading: rolesLoading } = useQuery({
    ...rolesQueryOptions(),
    enabled: open
  });
  const { data: juniors = EMPTY_ROLES, isLoading: hierarchyLoading } = useQuery({
    ...roleHierarchyQueryOptions(role.code),
    enabled: open
  });

  useEffect(() => {
    if (!open) return;
    const next = juniors.map((junior) => junior.code);
    setSelected((current) =>
      current.length === next.length && current.every((code, index) => code === next[index])
        ? current
        : next
    );
  }, [juniors, open]);

  const mutation = useMutation({
    ...mergeMutationOptions(setRoleHierarchyMutation, {
      onSuccess: () => {
        toast.success(tm('hierarchyUpdated'));
        onOpenChange(false);
      },
      onError: () => toast.error(tm('hierarchyUpdateFailed'))
    })
  });

  const candidates = allRoles.filter((candidate) => candidate.code !== role.code);
  const toggle = (code: string) => {
    if (!canEdit) return;
    setSelected((current) =>
      current.includes(code) ? current.filter((item) => item !== code) : [...current, code]
    );
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{t('title', { role: ts(role.name) })}</SheetTitle>
          <SheetDescription>{t('description')}</SheetDescription>
        </SheetHeader>
        <div className='flex-1 space-y-2 overflow-auto px-1'>
          {rolesLoading || hierarchyLoading ? (
            <div className='flex justify-center py-10'>
              <Icons.spinner className='size-6 animate-spin' />
            </div>
          ) : (
            candidates.map((candidate) => (
              <label
                key={candidate.code}
                htmlFor={`junior-role-${candidate.id}`}
                className='flex cursor-pointer items-start gap-3 rounded-md border p-3'
              >
                <Checkbox
                  id={`junior-role-${candidate.id}`}
                  checked={selected.includes(candidate.code)}
                  disabled={!canEdit || !candidate.isEnabled}
                  onCheckedChange={() => toggle(candidate.code)}
                />
                <span className='min-w-0'>
                  <span className='block text-sm font-medium'>{ts(candidate.name)}</span>
                  <code className='text-xs text-muted-foreground'>{candidate.code}</code>
                </span>
              </label>
            ))
          )}
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          {canEdit && (
            <Button
              isLoading={mutation.isPending}
              onClick={() => mutation.mutate({ code: role.code, juniorRoleCodes: selected })}
            >
              <Icons.check /> {tc('saveChanges')}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
