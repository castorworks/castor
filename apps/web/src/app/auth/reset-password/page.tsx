'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  resetAccountPasswordMutation,
  sendAuthCodeMutation
} from '@/features/account/api/mutations';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { checkPassword } from '@/lib/password-policy';
import { usePasswordPolicy } from '@/hooks/use-password-policy';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

const RESEND_COOLDOWN_SECONDS = 60;

export default function ResetPasswordPage() {
  const t = useTranslations('auth');
  const router = useRouter();
  const passwordPolicy = usePasswordPolicy();
  const [identifier, setIdentifier] = useState('');
  const [confirmCode, setConfirmCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [countdown, setCountdown] = useState(0);

  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setInterval(() => setCountdown((prev) => Math.max(0, prev - 1)), 1000);
    return () => clearInterval(timer);
  }, [countdown]);

  const sendCodeMutation = useMutation({
    ...mergeMutationOptions(sendAuthCodeMutation, {
      onSuccess: () => {
        setCountdown(RESEND_COOLDOWN_SECONDS);
        toast.success(t('verificationCodeSent'));
      },
      onError: () => toast.error(t('failedToSendCode'))
    })
  });

  const resetMutation = useMutation({
    ...mergeMutationOptions(resetAccountPasswordMutation, {
      onSuccess: () => {
        toast.success(t('passwordReset'));
        router.push('/auth/sign-in');
      },
      onError: (error) => toast.error(error.message || t('failedToResetPassword'))
    })
  });

  const sendCode = () => {
    if (!identifier) {
      toast.error(t('enterEmailOrMobileFirst'));
      return;
    }

    sendCodeMutation.mutate({
      codeType: identifier.includes('@') ? 'EMAIL' : 'MOBILE',
      username: identifier,
      // A sign-in code must not be usable to reset a password
      purpose: 'reset'
    });
  };

  const submit = () => {
    if (!identifier || !confirmCode || !newPassword) {
      toast.error(t('resetRequired'));
      return;
    }
    // 在消耗验证码之前拒绝不合策略的密码（后端同样先校验再核销验证码）
    const issue = checkPassword(newPassword, passwordPolicy.policy);
    if (issue) {
      toast.error(passwordPolicy.messages[issue]);
      return;
    }

    resetMutation.mutate({ username: identifier, confirmCode, newPassword });
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
            <Label htmlFor='identifier'>{t('emailOrMobile')}</Label>
            <div className='flex gap-2'>
              <Input
                id='identifier'
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                placeholder={t('enterEmailOrMobile')}
                autoComplete='username'
              />
              <Button
                type='button'
                variant='outline'
                disabled={!identifier || countdown > 0 || sendCodeMutation.isPending}
                onClick={sendCode}
              >
                {countdown > 0 ? t('resendIn', { seconds: countdown }) : t('getCode')}
              </Button>
            </div>
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='code'>{t('verificationCode')}</Label>
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
            <p className='text-muted-foreground text-xs'>{passwordPolicy.hint}</p>
          </div>
          <Button disabled={resetMutation.isPending} onClick={submit}>
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
