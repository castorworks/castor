'use client';

import { useState } from 'react';
import { useAppForm, useFormFields } from '@/components/ui/tanstack-form';
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
import { createUserMutation, updateUserMutation } from '../api/mutations';
import type { User, UpdateUserPayload } from '../api/types';
import { toast } from 'sonner';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import {
  createUserSchema,
  updateUserSchema,
  type CreateUserFormValues,
  type UpdateUserFormValues
} from '../schemas/user';
import { usePasswordMinLength } from '../hooks/usePasswordMinLength';
import { encryptPassword } from '../lib/rsa';
import { FormDateTimeField } from './datetime-field';

interface UserFormSheetProps {
  user?: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UserFormSheet({ user, open, onOpenChange }: UserFormSheetProps) {
  const t = useTranslations('users');
  const tc = useTranslations('common');
  const isEdit = !!user;
  const minLength = usePasswordMinLength();

  const createMutation = useMutation({
    ...mergeMutationOptions(createUserMutation, {
      onSuccess: () => {
        toast.success(t('messages.createSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: () => toast.error(t('messages.createFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateUserMutation, {
      onSuccess: () => {
        toast.success(t('messages.updateSuccess'));
        onOpenChange(false);
      },
      onError: () => toast.error(t('messages.updateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: isEdit
      ? ({
          name: user?.name ?? '',
          avatar: user?.avatar ?? '',
          password: '',
          enable: user?.enable ?? true,
          locked: user?.locked ?? false,
          accountExpireDate: user?.accountExpireDate ?? null,
          credentialExpireDate: user?.credentialExpireDate ?? null
        } as UpdateUserFormValues)
      : ({
          username: '',
          name: '',
          password: ''
        } as CreateUserFormValues),
    validators: {
      onSubmit: isEdit ? updateUserSchema(minLength) : createUserSchema(minLength)
    },
    onSubmit: async ({ value }) => {
      if (isEdit) {
        const v = value as UpdateUserFormValues;
        const payload: UpdateUserPayload = {};
        if (v.name) payload.name = v.name;
        if (v.avatar) payload.avatar = v.avatar;
        if (v.password) {
          try {
            payload.password = await encryptPassword(v.password);
          } catch {
            toast.error(t('messages.encryptionFailed'));
            return;
          }
        }
        // Always include status fields in edit mode
        payload.enable = v.enable;
        payload.locked = v.locked;
        // Include expiry dates (null means clear)
        payload.accountExpireDate = v.accountExpireDate || null;
        payload.credentialExpireDate = v.credentialExpireDate || null;
        await updateMutation.mutateAsync({ id: user!.id, values: payload });
      } else {
        const v = value as CreateUserFormValues;
        let encryptedPassword: string;
        try {
          encryptedPassword = await encryptPassword(v.password);
        } catch {
          toast.error(t('messages.encryptionFailed'));
          return;
        }
        await createMutation.mutateAsync({
          username: v.username,
          name: v.name || undefined,
          password: encryptedPassword
        });
      }
    }
  });

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEdit ? t('form.editTitle') : t('form.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEdit ? t('form.editDescription') : t('form.newDescription')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='user-form-sheet' className='space-y-4'>
              {isEdit ? <UpdateFields /> : <CreateFields />}
            </form.Form>
          </form.AppForm>
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='user-form-sheet' isLoading={isPending}>
            <Icons.check /> {isEdit ? t('form.updateUser') : t('form.createUser')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// Create Form Fields (username, name, password)
// ============================================================

function CreateFields() {
  const t = useTranslations('users');
  const { FormTextField } = useFormFields<CreateUserFormValues>();

  return (
    <>
      <FormTextField name='username' label={t('form.username')} required placeholder='johndoe' />
      <FormTextField name='name' label={t('form.displayName')} placeholder='John Doe' />
      <FormTextField
        name='password'
        label={t('form.password')}
        required
        type='password'
        placeholder={t('form.setPassword')}
      />
    </>
  );
}

// ============================================================
// Update Form Fields (name, avatar, password, enable, locked, expiry dates)
// ============================================================

function UpdateFields() {
  const t = useTranslations('users');
  const { FormTextField, FormSwitchField } = useFormFields<UpdateUserFormValues>();

  return (
    <>
      <FormTextField name='name' label={t('form.displayName')} placeholder='John Doe' />
      <FormTextField
        name='avatar'
        label={t('form.avatarUrl')}
        type='url'
        placeholder='https://example.com/avatar.jpg'
      />
      <FormTextField
        name='password'
        label={t('form.newPasswordKeep')}
        type='password'
        placeholder={t('form.newPasswordPlaceholder')}
      />

      {/* Status toggles */}
      <FormSwitchField
        name='enable'
        label={t('form.enableAccount')}
        description={t('form.enableAccountDescription')}
      />
      <FormSwitchField
        name='locked'
        label={t('form.lockAccount')}
        description={t('form.lockAccountDescription')}
      />

      {/* Expiry date fields */}
      <FormDateTimeField
        name='accountExpireDate'
        label={t('form.accountExpireDate')}
        clearLabel={t('form.clearExpireDate')}
      />
      <FormDateTimeField
        name='credentialExpireDate'
        label={t('form.credentialExpireDate')}
        clearLabel={t('form.clearExpireDate')}
      />
    </>
  );
}

// ============================================================
// Trigger Button
// ============================================================

export function UserFormSheetTrigger() {
  const t = useTranslations('users');
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.add className='mr-2 h-4 w-4' /> {t('list.addUser')}
      </Button>
      <UserFormSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
