'use client';

import { useState } from 'react';
import { typedField, useAppForm, useFormFields } from '@/components/ui/tanstack-form';
import { FormAssetField } from '@/components/forms/fields';
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
  NO_DEPARTMENT,
  createUserSchema,
  updateUserSchema,
  type CreateUserFormValues,
  type UpdateUserFormValues
} from '../schemas/user';
import { useDepartmentOptions } from '../hooks/use-department-options';
import { usePasswordPolicy } from '@/hooks/use-password-policy';
import { encryptPassword } from '../lib/rsa';
import { FormDateTimeField } from './datetime-field';

const FormAvatarField = typedField<UpdateUserFormValues>()(FormAssetField);

/** Form value → department id; the sentinel (or an empty value) means no department. */
function departmentIdOf(value: string | undefined): number | null {
  return value && value !== NO_DEPARTMENT ? Number(value) : null;
}

interface UserFormSheetProps {
  user?: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UserFormSheet({ user, open, onOpenChange }: UserFormSheetProps) {
  const t = useTranslations('users');
  const tc = useTranslations('common');
  const isEdit = !!user;
  const { policy, messages: passwordMessages } = usePasswordPolicy();
  const formMessages = {
    usernameTooShort: t('form.usernameTooShort'),
    invalidEmail: t('form.invalidEmail'),
    invalidMobile: t('form.invalidMobile'),
    password: passwordMessages
  };

  const createMutation = useMutation({
    ...mergeMutationOptions(createUserMutation, {
      onSuccess: () => {
        toast.success(t('messages.createSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: (error) => toast.error(error.message || t('messages.createFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateUserMutation, {
      onSuccess: () => {
        toast.success(t('messages.updateSuccess'));
        onOpenChange(false);
      },
      onError: (error) => toast.error(error.message || t('messages.updateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: isEdit
      ? ({
          name: user?.name ?? '',
          avatar: user?.avatar ?? '',
          email: user?.email ?? '',
          mobile: user?.mobile ?? '',
          password: '',
          department: user?.departmentId ? String(user.departmentId) : NO_DEPARTMENT,
          enable: user?.enable ?? true,
          locked: user?.locked ?? false,
          accountExpireDate: user?.accountExpireDate ?? null,
          credentialExpireDate: user?.credentialExpireDate ?? null
        } as UpdateUserFormValues)
      : ({
          username: '',
          name: '',
          password: '',
          email: '',
          mobile: '',
          department: NO_DEPARTMENT
        } as CreateUserFormValues),
    validators: {
      onSubmit: isEdit
        ? updateUserSchema(formMessages, policy)
        : createUserSchema(formMessages, policy)
    },
    onSubmit: async ({ value }) => {
      if (isEdit) {
        const v = value as UpdateUserFormValues;
        const payload: UpdateUserPayload = {};
        if (v.name) payload.name = v.name;
        // 头像是资产 objectKey；只在更换时提交，后端据此登记引用并回收旧头像。
        if (v.avatar && v.avatar !== (user?.avatar ?? '')) payload.avatar = v.avatar;
        // Only send contacts the admin actually changed ('' explicitly clears one)
        if ((v.email ?? '') !== (user?.email ?? '')) payload.email = v.email ?? '';
        if ((v.mobile ?? '') !== (user?.mobile ?? '')) payload.mobile = v.mobile ?? '';
        const department = departmentIdOf(v.department);
        if (department !== (user?.departmentId ?? null)) payload.departmentId = department ?? 0;
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
          password: encryptedPassword,
          email: v.email || undefined,
          mobile: v.mobile || undefined,
          departmentId: departmentIdOf(v.department)
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
  const { hint } = usePasswordPolicy();
  const departments = useDepartmentOptions();
  const { FormTextField, FormSelectField } = useFormFields<CreateUserFormValues>();

  return (
    <>
      <FormTextField name='username' label={t('form.username')} required placeholder='johndoe' />
      <FormTextField
        name='name'
        label={t('form.displayName')}
        placeholder={t('form.displayNamePlaceholder')}
      />
      <FormTextField
        name='password'
        label={t('form.password')}
        required
        type='password'
        placeholder={t('form.setPassword')}
        description={hint}
      />
      <FormTextField
        name='email'
        label={t('form.email')}
        type='email'
        placeholder={t('form.emailPlaceholder')}
      />
      <FormTextField
        name='mobile'
        label={t('form.mobile')}
        type='tel'
        placeholder={t('form.mobilePlaceholder')}
      />
      {departments.available && (
        <FormSelectField
          name='department'
          label={t('form.department')}
          description={t('form.departmentDescription')}
          options={departments.options}
        />
      )}
    </>
  );
}

// ============================================================
// Update Form Fields (name, avatar, password, enable, locked, expiry dates)
// ============================================================

function UpdateFields() {
  const t = useTranslations('users');
  const { hint } = usePasswordPolicy();
  const departments = useDepartmentOptions();
  const { FormTextField, FormSwitchField, FormSelectField } = useFormFields<UpdateUserFormValues>();

  return (
    <>
      <FormTextField
        name='name'
        label={t('form.displayName')}
        placeholder={t('form.displayNamePlaceholder')}
      />
      <FormAvatarField
        name='avatar'
        label={t('form.avatar')}
        description={t('form.avatarDescription')}
        category='IMAGE'
        accept='image/png,image/jpeg,image/gif,image/webp'
        isPublic
      />
      <FormTextField
        name='email'
        label={t('form.email')}
        type='email'
        placeholder={t('form.emailPlaceholder')}
      />
      <FormTextField
        name='mobile'
        label={t('form.mobile')}
        type='tel'
        placeholder={t('form.mobilePlaceholder')}
      />
      {departments.available && (
        <FormSelectField
          name='department'
          label={t('form.department')}
          description={t('form.departmentDescription')}
          options={departments.options}
        />
      )}
      <FormTextField
        name='password'
        label={t('form.newPasswordKeep')}
        type='password'
        placeholder={t('form.newPasswordPlaceholder')}
        description={hint}
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
        description={t('form.credentialExpireDateDescription')}
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
