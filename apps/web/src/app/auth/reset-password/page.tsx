'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { toast } from 'sonner';
import { apiClient } from '@/lib/api-client';
import { encryptPassword } from '@/lib/rsa';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

type PublicKeyResponse = {
  publicKey: string;
};

export default function ResetPasswordPage() {
  const t = useTranslations('auth');
  const tc = useTranslations('common');
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [confirmCode, setConfirmCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [isSendingCode, setIsSendingCode] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const sendCode = async () => {
    if (!username) {
      toast.error(t('enterUsernameFirst'));
      return;
    }

    setIsSendingCode(true);
    try {
      await apiClient('/v1/auth/code', {
        method: 'POST',
        body: JSON.stringify({
          codeType: username.includes('@') ? 'EMAIL' : 'MOBILE',
          username
        })
      });
      toast.success(t('verificationCodeSent'));
    } catch {
      toast.error(t('failedToSendCode'));
    } finally {
      setIsSendingCode(false);
    }
  };

  const submit = async () => {
    if (!username || !confirmCode || !newPassword) {
      toast.error(t('resetRequired'));
      return;
    }

    setIsSubmitting(true);
    try {
      const { publicKey } = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
      const encryptedPassword = encryptPassword(publicKey, newPassword);
      await apiClient('/v1/account/password/reset', {
        method: 'PUT',
        body: JSON.stringify({
          username,
          confirmCode,
          newPassword: encryptedPassword
        })
      });
      toast.success(t('passwordReset'));
      router.push('/auth/sign-in');
    } catch {
      toast.error(t('failedToResetPassword'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className='flex min-h-screen items-center justify-center bg-muted/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <CardTitle className='text-2xl'>{t('resetPasswordPageTitle')}</CardTitle>
          <CardDescription>{t('resetPasswordPageSubtitle')}</CardDescription>
        </CardHeader>
        <CardContent className='flex flex-col gap-4'>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='username'>{t('usernameIdentifier')}</Label>
            <div className='flex gap-2'>
              <Input
                id='username'
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete='username'
              />
              <Button type='button' variant='outline' disabled={isSendingCode} onClick={sendCode}>
                {tc('send')}
              </Button>
            </div>
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='code'>{tc('verificationCode')}</Label>
            <Input
              id='code'
              value={confirmCode}
              onChange={(event) => setConfirmCode(event.target.value)}
              maxLength={10}
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='new-password'>{t('newPassword')}</Label>
            <Input
              id='new-password'
              type='password'
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              autoComplete='new-password'
            />
          </div>
          <Button disabled={isSubmitting} onClick={submit}>
            {t('resetPassword')}
          </Button>
          <p className='text-center text-sm text-muted-foreground'>
            {t('rememberedPassword')}{' '}
            <Link href='/auth/sign-in' className='text-primary underline hover:text-primary/80'>
              {t('signIn')}
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
