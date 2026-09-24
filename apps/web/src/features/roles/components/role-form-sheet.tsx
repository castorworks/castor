'use client';

import { useState, useEffect } from 'react';
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
import { useTranslations } from 'next-intl';
import { createRoleMutation, updateRoleMutation } from '../api/mutations';
import { toast } from 'sonner';
import {
  createRoleSchema,
  updateRoleSchema,
  type CreateRoleFormValues,
  type UpdateRoleFormValues
} from '../schemas/role';
import type { Role } from '../api/types';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface RoleFormSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  role?: Role;
}

export function RoleFormSheet({ open, onOpenChange, role }: RoleFormSheetProps) {
  const t = useTranslations('roles');
  const tc = useTranslations('common');
  const isEditing = !!role;

  const createMutation = useMutation({
    ...mergeMutationOptions(createRoleMutation, {
      onSuccess: () => {
        toast.success(t('messages.createSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.createFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateRoleMutation, {
      onSuccess: () => {
        toast.success(t('messages.updateSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.updateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: isEditing
      ? ({ name: role.name, description: role.description ?? '' } as UpdateRoleFormValues)
      : ({
          code: '',
          name: '',
          description: ''
        } as CreateRoleFormValues),
    validators: {
      onSubmit: isEditing ? updateRoleSchema : createRoleSchema
    },
    onSubmit: async ({ value }) => {
      if (isEditing) {
        await updateMutation.mutateAsync({
          code: role.code,
          data: {
            name: (value as UpdateRoleFormValues).name,
            description: (value as UpdateRoleFormValues).description || undefined
          }
        });
      } else {
        await createMutation.mutateAsync({
          code: (value as CreateRoleFormValues).code,
          name: (value as CreateRoleFormValues).name,
          description: (value as CreateRoleFormValues).description || undefined
        });
      }
    }
  });

  // Reset form values when sheet opens or role changes
  useEffect(() => {
    if (open) {
      if (isEditing) {
        form.reset({
          name: role.name,
          description: role.description ?? ''
        } as UpdateRoleFormValues);
      } else {
        form.reset({ code: '', name: '', description: '' } as CreateRoleFormValues);
      }
    }
  }, [form, isEditing, open, role?.code, role?.description, role?.name]);

  const isCreateMode = !isEditing;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEditing ? t('form.editTitle') : t('form.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEditing ? t('form.editDescription') : t('form.newDescription')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='role-form-sheet' className='space-y-4'>
              {isCreateMode && (
                <form.TextField name='code' label={tc('code')} required placeholder='manager' />
              )}
              <form.TextField
                name='name'
                label={tc('name')}
                required
                placeholder={t('form.managerPlaceholder')}
              />
              <form.TextField
                name='description'
                label={tc('description')}
                placeholder={t('form.describePlaceholder')}
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
            form='role-form-sheet'
            isLoading={createMutation.isPending || updateMutation.isPending}
          >
            <Icons.check /> {isEditing ? tc('saveChanges') : t('form.createRole')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

export function RoleFormSheetTrigger() {
  const t = useTranslations('roles');
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.add className='mr-2 h-4 w-4' /> {t('list.addRole')}
      </Button>
      <RoleFormSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
