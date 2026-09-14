'use client';

import { useRef } from 'react';
import { useSuspenseQuery, useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';
import { Icons } from '@/components/icons';
import { accountInfoQueryOptions, userRolesQueryOptions } from '../api/queries';
import {
  updateAccountNameMutation,
  updateAccountPasswordMutation,
  uploadAccountAvatarMutation
} from '../api/mutations';
import {
  updateNameSchema,
  updatePasswordSchema,
  type UpdateNameFormValues,
  type UpdatePasswordFormValues
} from '../schemas/account';
import { useAppForm } from '@/components/ui/tanstack-form';
import { useAuthStore } from '@/stores/auth-store';
import { getDictLabel } from '@/lib/dict';
import { useTranslations } from 'next-intl';
import { useFormat } from '@/hooks/use-format';
import { mergeMutationOptions } from '@/lib/mutation-utils';

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

export default function ProfilePage() {
  const t = useTranslations('profile');
  const tc = useTranslations('common');
  const ta = useTranslations('auth');
  const fmt = useFormat();
  const { data: accountInfo } = useSuspenseQuery(accountInfoQueryOptions());
  const { data: rolesData } = useSuspenseQuery(userRolesQueryOptions());

  const nameMutation = useMutation({
    ...mergeMutationOptions(updateAccountNameMutation, {
      onSuccess: () => toast.success(t('messages.nameUpdated')),
      onError: () => toast.error(t('messages.nameUpdateFailed'))
    })
  });

  const logout = useAuthStore((s) => s.logout);

  const passwordMutation = useMutation({
    ...mergeMutationOptions(updateAccountPasswordMutation, {
      onSuccess: () => {
        // 后端改密后会撤销全部会话，需重新登录
        toast.success(t('messages.passwordUpdated'));
        void logout();
      },
      onError: () => toast.error(t('messages.passwordUpdateFailed'))
    })
  });

  const avatarMutation = useMutation({
    ...mergeMutationOptions(uploadAccountAvatarMutation, {
      onSuccess: () => {
        toast.success(t('messages.avatarUpdated'));
      },
      onError: () => toast.error(t('messages.avatarUpdateFailed'))
    })
  });

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleAvatarClick = () => {
    fileInputRef.current?.click();
  };

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (file.size > 2 * 1024 * 1024) {
      toast.error(t('messages.imageTooLarge'));
      return;
    }

    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      toast.error(t('messages.invalidImageType'));
      return;
    }

    const formData = new FormData();
    formData.append('file', file);
    avatarMutation.mutate(formData);
  };

  const roles = rolesData?.activeRoles ?? [];
  const accountSourceLabel = getDictLabel('account_source', accountInfo.accountSource);
  const accountDetails = [
    { label: ta('username'), value: accountInfo.username },
    { label: t('accountSource'), value: accountSourceLabel },
    { label: tc('created'), value: fmt.date(accountInfo.createdAt) },
    { label: t('lastUpdated'), value: fmt.date(accountInfo.updatedAt) }
  ];

  return (
    <div className='flex flex-col gap-4'>
      <Card className='gap-0 overflow-hidden'>
        <CardContent className='flex flex-col gap-5 py-5 lg:flex-row lg:items-center lg:justify-between'>
          <div className='flex min-w-0 flex-col gap-4 sm:flex-row sm:items-center'>
            <div className='group relative shrink-0'>
              <Avatar
                className='size-16 cursor-pointer ring-1 ring-border'
                onClick={handleAvatarClick}
              >
                <AvatarImage
                  src={
                    accountInfo.avatar
                      ? `/api/${accountInfo.avatar}?t=${encodeURIComponent(accountInfo.updatedAt)}`
                      : undefined
                  }
                  alt={accountInfo.name}
                />
                <AvatarFallback className='text-lg font-medium'>
                  {getInitials(accountInfo.name)}
                </AvatarFallback>
              </Avatar>
              <div className='bg-muted/80 absolute inset-0 flex items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-100'>
                <Icons.edit className='text-muted-foreground' />
              </div>
            </div>
            <input
              ref={fileInputRef}
              type='file'
              accept='image/jpeg,image/png,image/gif,image/webp'
              className='hidden'
              onChange={handleAvatarChange}
            />
            <div className='flex min-w-0 flex-1 flex-col gap-2.5'>
              <div className='min-w-0'>
                <h3 className='truncate text-xl font-semibold tracking-tight'>
                  {accountInfo.name}
                </h3>
                <p className='text-muted-foreground truncate text-sm'>{accountInfo.username}</p>
              </div>
              <div className='flex flex-wrap items-center gap-2'>
                <Badge variant='outline'>{accountSourceLabel}</Badge>
                {roles.map((role) => (
                  <Badge key={role.code} variant='secondary'>
                    {role.name}
                  </Badge>
                ))}
              </div>
            </div>
            {avatarMutation.isPending && (
              <div>
                <Icons.spinner className='animate-spin text-muted-foreground' />
              </div>
            )}
          </div>
          <dl className='grid gap-3 sm:grid-cols-2 lg:min-w-[620px] lg:grid-cols-4'>
            {accountDetails.map((item) => (
              <div key={item.label} className='min-w-0 rounded-md bg-muted/20 px-3 py-2'>
                <dt className='text-muted-foreground text-xs'>{item.label}</dt>
                <dd className='truncate text-sm font-medium'>{item.value}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      <div className='grid items-start gap-4 xl:grid-cols-[minmax(320px,0.8fr)_minmax(520px,1fr)]'>
        <Card className='gap-4 self-start'>
          <CardHeader className='pb-0'>
            <CardTitle>{ta('displayName')}</CardTitle>
            <CardDescription>{t('displayNameDescription')}</CardDescription>
          </CardHeader>
          <CardContent className='max-w-lg'>
            <NameForm
              currentName={accountInfo.name}
              onSubmit={(values) => nameMutation.mutate(values)}
              isPending={nameMutation.isPending}
            />
          </CardContent>
        </Card>

        <Card className='gap-4'>
          <CardHeader className='pb-0'>
            <CardTitle>{ta('password')}</CardTitle>
            <CardDescription>{t('passwordDescription')}</CardDescription>
          </CardHeader>
          <CardContent className='max-w-xl'>
            <PasswordForm
              onSubmit={(values) => passwordMutation.mutate(values)}
              isPending={passwordMutation.isPending}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Name Form
// ---------------------------------------------------------------------------

interface NameFormProps {
  currentName: string;
  onSubmit: (values: UpdateNameFormValues) => void;
  isPending: boolean;
}

function NameForm({ currentName, onSubmit, isPending }: NameFormProps) {
  const t = useTranslations('profile');
  const tc = useTranslations('common');
  const ta = useTranslations('auth');
  const form = useAppForm({
    defaultValues: { name: currentName } as UpdateNameFormValues,
    validators: { onSubmit: updateNameSchema },
    onSubmit: ({ value }) => onSubmit(value as UpdateNameFormValues)
  });

  return (
    <form.AppForm>
      <form.Form id='name-form' className='p-0 md:p-0'>
        <div className='flex max-w-md flex-col gap-3'>
          <form.TextField
            name='name'
            label={ta('displayName')}
            required
            placeholder={t('yourName')}
          />
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={() => form.setFieldValue('name', currentName)}
            >
              {tc('reset')}
            </Button>
            <Button
              type='submit'
              form='name-form'
              size='sm'
              isLoading={isPending}
              disabled={isPending}
            >
              <Icons.check data-icon='inline-start' /> {tc('save')}
            </Button>
          </div>
        </div>
      </form.Form>
    </form.AppForm>
  );
}

// ---------------------------------------------------------------------------
// Password Form
// ---------------------------------------------------------------------------

interface PasswordFormProps {
  onSubmit: (values: UpdatePasswordFormValues) => void;
  isPending: boolean;
}

function PasswordForm({ onSubmit, isPending }: PasswordFormProps) {
  const t = useTranslations('profile');
  const ta = useTranslations('auth');
  const form = useAppForm({
    defaultValues: {
      currentPassword: '',
      newPassword: '',
      confirmPassword: ''
    } as UpdatePasswordFormValues,
    validators: { onSubmit: updatePasswordSchema },
    onSubmit: ({ value }) => onSubmit(value as UpdatePasswordFormValues)
  });

  return (
    <form.AppForm>
      <form.Form id='password-form' className='p-0 md:p-0'>
        <div className='flex max-w-lg flex-col gap-3'>
          <form.TextField
            name='currentPassword'
            label={t('currentPassword')}
            type='password'
            required
            placeholder={t('enterCurrentPassword')}
          />
          <form.TextField
            name='newPassword'
            label={t('newPassword')}
            type='password'
            required
            placeholder={t('atLeastEightCharacters')}
          />
          <form.TextField
            name='confirmPassword'
            label={ta('confirmPassword')}
            type='password'
            required
            placeholder={t('reenterNewPassword')}
          />
        </div>
        <div className='flex max-w-lg flex-col gap-3'>
          <Separator />
          <Button
            className='self-end'
            type='submit'
            form='password-form'
            size='sm'
            isLoading={isPending}
            disabled={isPending}
          >
            <Icons.lock data-icon='inline-start' /> {t('updatePassword')}
          </Button>
        </div>
      </form.Form>
    </form.AppForm>
  );
}
