'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { useDict } from '@/hooks/use-dict';
import { usePermission } from '@/hooks/use-permission';
import { dictOptionsOr } from '@/lib/dict';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { departmentsQueryOptions } from '@/features/departments/api/queries';
import { flattenDepartments } from '@/features/departments/tree';
import { setRoleDataScopeMutation } from '../api/mutations';
import { DATA_SCOPES, type DataScope, type Role } from '../api/types';

interface RoleDataScopeSheetProps {
  role: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Chooses which users' data members of the role can see. System roles are seed
 * managed and shown read-only; the backend also refuses scopes wider than the
 * caller's own (message shown as returned).
 */
export function RoleDataScopeSheet({ role, open, onOpenChange }: RoleDataScopeSheetProps) {
  const t = useTranslations('roles.dataScope');
  const tc = useTranslations('common');
  const { ts } = useSeedTranslation();
  const dict = useDict();
  const canEdit = usePermission('/api/v1/admin/roles/:role/data-scope:PUT') && !role.isSystem;
  const canListDepartments = usePermission('/api/v1/admin/departments:GET');
  const { data: departments = [] } = useQuery({
    ...departmentsQueryOptions(),
    enabled: open && canListDepartments
  });
  const [scope, setScope] = useState<DataScope>(role.dataScope);
  const [selected, setSelected] = useState<number[]>(role.departmentIds ?? []);

  useEffect(() => {
    if (!open) return;
    setScope(role.dataScope);
    setSelected(role.departmentIds ?? []);
  }, [open, role.dataScope, role.departmentIds]);

  const mutation = useMutation({
    ...mergeMutationOptions(setRoleDataScopeMutation, {
      onSuccess: () => {
        toast.success(t('saved'));
        onOpenChange(false);
      },
      onError: (error) => toast.error(error.message || t('saveFailed'))
    })
  });

  const options = dictOptionsOr(
    dict,
    'role_data_scope',
    DATA_SCOPES.map((value) => ({ value, label: value }))
  );
  const toggle = (id: number) =>
    setSelected((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id]
    );
  const customInvalid = scope === 'CUSTOM' && selected.length === 0;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{t('title', { role: ts(role.name) })}</SheetTitle>
          <SheetDescription>
            {role.isSystem ? t('systemReadOnly') : t('description')}
          </SheetDescription>
        </SheetHeader>
        <div className='flex-1 space-y-4 overflow-auto px-1'>
          <RadioGroup
            value={scope}
            onValueChange={(value) => setScope(value as DataScope)}
            disabled={!canEdit}
            className='gap-2'
          >
            {options.map((option) => (
              <label
                key={option.value}
                htmlFor={`data-scope-${option.value}`}
                className='flex cursor-pointer items-start gap-3 rounded-md border p-3'
              >
                <RadioGroupItem id={`data-scope-${option.value}`} value={option.value} />
                <span className='min-w-0'>
                  <span className='block text-sm font-medium'>{option.label}</span>
                  <span className='text-muted-foreground block text-xs'>
                    {t(`hints.${option.value as DataScope}`)}
                  </span>
                </span>
              </label>
            ))}
          </RadioGroup>
          {scope === 'CUSTOM' && (
            <div className='space-y-2'>
              <Label>{t('departments')}</Label>
              {!canListDepartments ? (
                <p className='text-muted-foreground text-sm'>{t('noDepartmentAccess')}</p>
              ) : departments.length === 0 ? (
                <p className='text-muted-foreground text-sm'>{t('noDepartments')}</p>
              ) : (
                <div className='space-y-1 rounded-md border p-2'>
                  {flattenDepartments(departments).map(({ department, depth }) => (
                    <label
                      key={department.id}
                      htmlFor={`scope-department-${department.id}`}
                      className='flex cursor-pointer items-center gap-2 rounded px-1 py-1 text-sm'
                      style={{ paddingLeft: depth * 16 + 4 }}
                    >
                      <Checkbox
                        id={`scope-department-${department.id}`}
                        checked={selected.includes(department.id)}
                        disabled={!canEdit || !department.inScope}
                        onCheckedChange={() => toggle(department.id)}
                      />
                      <span className={department.inScope ? '' : 'text-muted-foreground'}>
                        {department.name}
                      </span>
                    </label>
                  ))}
                </div>
              )}
              {customInvalid && <p className='text-destructive text-xs'>{t('selectAtLeastOne')}</p>}
            </div>
          )}
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          {canEdit && (
            <Button
              isLoading={mutation.isPending}
              disabled={customInvalid}
              onClick={() =>
                mutation.mutate({ code: role.code, dataScope: scope, departmentIds: selected })
              }
            >
              <Icons.check /> {tc('saveChanges')}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
