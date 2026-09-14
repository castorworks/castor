'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { apiClient } from '@/lib/api-client';
import { encryptPassword } from '@/lib/rsa';
import type { CaptchaResponse } from '@/stores/auth-store';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

type PublicKeyResponse = {
  publicKey: string;
};

function toDataUri(img: string) {
  return img.startsWith('data:') ? img : `data:image/png;base64,${img}`;
}

export default function SignUpPage() {
  const t = useTranslations('auth');
  const tc = useTranslations('common');
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [captchaImg, setCaptchaImg] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const fetchCaptcha = useCallback(async () => {
    try {
      const res = await apiClient<CaptchaResponse>('/v1/auth/captcha');
      setCaptchaId(res.id);
      setCaptchaImg(toDataUri(res.img));
    } catch {
      toast.error(t('failedToLoadCaptcha'));
    }
  }, [t]);

  useEffect(() => {
    fetchCaptcha();
  }, [fetchCaptcha]);

  const submit = async () => {
    if (!username || !password || !captchaId || !captchaCode) {
      toast.error(t('registrationRequired'));
      return;
    }

    setIsSubmitting(true);
    try {
      const { publicKey } = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
      const encryptedPassword = encryptPassword(publicKey, password);
      await apiClient('/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify({
          username,
          name: name || undefined,
          password: encryptedPassword,
          captchaId,
          captchaCode
        })
      });
      toast.success(t('accountCreated'));
      router.push('/auth/sign-in');
    } catch {
      toast.error(t('registrationFailed'));
      fetchCaptcha();
      setCaptchaCode('');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className='flex min-h-screen items-center justify-center bg-muted/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <CardTitle className='text-2xl'>{t('createAccountTitle')}</CardTitle>
          <CardDescription>{t('createAccountSubtitle')}</CardDescription>
        </CardHeader>
        <CardContent className='flex flex-col gap-4'>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='username'>{t('username')}</Label>
            <Input
              id='username'
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder={t('enterUsername')}
              autoComplete='username'
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='name'>{t('displayName')}</Label>
            <Input
              id='name'
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder={tc('optional')}
              autoComplete='name'
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='password'>{t('password')}</Label>
            <Input
              id='password'
              type='password'
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder={t('atLeastSixCharacters')}
              autoComplete='new-password'
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='captcha'>{t('captcha')}</Label>
            <div className='flex gap-2'>
              <Input
                id='captcha'
                value={captchaCode}
                onChange={(event) => setCaptchaCode(event.target.value)}
                placeholder={t('captchaPlaceholder')}
                maxLength={10}
              />
              {captchaImg && (
                <button
                  type='button'
                  className='h-9 overflow-hidden rounded-md border border-slate-200 bg-white p-0.5 dark:border-slate-300 dark:bg-slate-100'
                  onClick={fetchCaptcha}
                  title={t('clickToRefresh')}
                  aria-label={t('refreshCaptcha')}
                >
                  <Image
                    src={captchaImg}
                    alt={t('captchaAlt')}
                    width={96}
                    height={36}
                    unoptimized
                    className='h-8 w-[5.75rem] rounded-[3px] object-cover'
                  />
                </button>
              )}
            </div>
          </div>
          <Button disabled={isSubmitting} onClick={submit}>
            {tc('createAccount')}
          </Button>
          <p className='text-center text-sm text-muted-foreground'>
            {t('alreadyHaveAccount')}{' '}
            <Link href='/auth/sign-in' className='text-primary underline hover:text-primary/80'>
              {t('signIn')}
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
