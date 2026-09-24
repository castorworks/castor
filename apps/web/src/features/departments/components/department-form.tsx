'use client';

import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
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
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { saveDepartmentMutation } from '../api/mutations';
import type { Department } from '../api/types';
import { departmentSchema, type DepartmentFormValues } from '../schemas/department';
import { flattenDepartments, indentedLabel, subtreeIds } from '../tree';

const ROOT = 'root';

interface DepartmentFormProps {
  department?: Department;
  /** Pre-selected parent when adding a child from the tree. */
  parentId?: number;
  departments: Department[];
  /** Only callers with an ALL data scope may create or move departments at the root. */
  canUseRoot: boolean;
  onClose: () => void;
}

export function DepartmentForm({
  department,
  parentId,
  departments,
  canUseRoot,
  onClose
}: DepartmentFormProps) {
  const t = useTranslations('departments');
  const tc = useTranslations('common');
  const mutation = useMutation({
    ...mergeMutationOptions(saveDepartmentMutation, {
      onSuccess: () => {
        toast.success(t('messages.saved'));
        onClose();
      },
      onError: (error) => toast.error(error.message || t('messages.saveFailed'))
    })
  });

  // 不能挂到自己或自己的后代下面；只能挂到数据范围内的部门。
  const excluded = new Set(department ? subtreeIds(departments, department.id) : []);
  const parentOptions = [
    ...(canUseRoot ? [{ value: ROOT, label: t('form.root') }] : []),
    ...flattenDepartments(departments)
      .filter(({ department: d }) => d.inScope && !excluded.has(d.id))
      .map(({ department: d, depth }) => ({
        value: String(d.id),
        label: indentedLabel(d.name, depth)
      }))
  ];
  const initialParent = department ? department.parentId : (parentId ?? null);

  const form = useAppForm({
    defaultValues: {
      parentId: initialParent === null ? ROOT : String(initialParent),
      code: department?.code ?? '',
      name: department?.name ?? '',
      sortOrder: department?.sortOrder ?? 0,
      isEnabled: department?.isEnabled ?? true
    } as DepartmentFormValues,
    validators: {
      onSubmit: departmentSchema({
        codeInvalid: t('validation.codeInvalid'),
        nameRequired: t('validation.nameRequired')
      })
    },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        id: department?.id,
        data: {
          parentId: value.parentId === ROOT ? null : Number(value.parentId),
          code: value.code.trim(),
          name: value.name.trim(),
          sortOrder: Number(value.sortOrder) || 0,
          isEnabled: value.isEnabled
        }
      });
    }
  });

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className='flex w-full flex-col sm:max-w-lg'>
        <SheetHeader>
          <SheetTitle>{department ? t('form.editTitle') : t('form.createTitle')}</SheetTitle>
          <SheetDescription>{t('form.description')}</SheetDescription>
        </SheetHeader>
        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='department-form' className='space-y-4'>
              <form.SelectField name='parentId' label={t('form.parent')} options={parentOptions} />
              <form.TextField name='name' label={t('form.name')} required />
              <form.TextField
                name='code'
                label={t('form.code')}
                description={t('form.codeDescription')}
                required
              />
              <form.TextField name='sortOrder' label={t('form.sortOrder')} type='number' />
              <form.SwitchField
                name='isEnabled'
                label={t('form.enabled')}
                description={t('form.enabledDescription')}
              />
            </form.Form>
          </form.AppForm>
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={onClose}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='department-form' isLoading={mutation.isPending}>
            <Icons.check /> {tc('save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
