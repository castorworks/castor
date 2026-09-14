// ============================================================
// Sign-In Page
// ============================================================
// Supports three login methods: password, email code, mobile code.
// Integrates with castor JWT authentication.
// ============================================================

'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore, type CaptchaResponse } from '@/stores/auth-store';
import { apiClient, CastorApiError } from '@/lib/api-client';
import { safeRedirectPath } from '@/lib/safe-redirect';
import { publicSettingsQueryOptions } from '@/features/settings/api/queries';
import type { LoginMethod } from '@/features/settings/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Icons } from '@/components/icons';
import { toast } from 'sonner';

// Backend error code for captcha required/invalid
const ERR_CODE_INVALID_CAPTCHA = 2002;
const LOGIN_METHODS_SETTING_KEY = 'security.login.allowedMethods';
const DEFAULT_LOGIN_METHODS: LoginMethod[] = ['password', 'email', 'mobile'];

function resolveAllowedLoginMethods(value: unknown): LoginMethod[] {
  if (!Array.isArray(value)) {
    return DEFAULT_LOGIN_METHODS;
  }

  const allowed = DEFAULT_LOGIN_METHODS.filter((method) => value.includes(method));
  return allowed.length > 0 ? allowed : DEFAULT_LOGIN_METHODS;
}

function toDataUri(img: string) {
  return img.startsWith('data:') ? img : `data:image/png;base64,${img}`;
}

function CaptchaImage({
  src,
  onRefresh,
  refreshTitle,
  refreshLabel,
  alt
}: {
  src: string;
  onRefresh: () => void;
  refreshTitle: string;
  refreshLabel: string;
  alt: string;
}) {
  return (
    <button
      type='button'
      className='h-9 overflow-hidden rounded-md border border-slate-200 bg-white p-0.5 dark:border-slate-300 dark:bg-slate-100'
      onClick={onRefresh}
      title={refreshTitle}
      aria-label={refreshLabel}
    >
      <Image
        src={src}
        alt={alt}
        width={96}
        height={36}
        unoptimized
        className='h-8 w-23 rounded-[3px] object-cover'
      />
    </button>
  );
}

export default function SignInPage() {
  const t = useTranslations('auth');
  const tc = useTranslations('common');
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirectPath = safeRedirectPath(searchParams.get('redirect'));

  const { login, isSubmitting, error, clearError } = useAuthStore();
  const { data: publicSettings } = useQuery(publicSettingsQueryOptions());
  const allowedLoginMethods = useMemo(
    () => resolveAllowedLoginMethods(publicSettings?.[LOGIN_METHODS_SETTING_KEY]),
    [publicSettings]
  );

  // Password tab state
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [captchaImg, setCaptchaImg] = useState('');
  const [captchaRequired, setCaptchaRequired] = useState(false);
  const [rememberMe, setRememberMe] = useState(false);

  // Email tab state
  const [email, setEmail] = useState('');
  const [emailCode, setEmailCode] = useState('');
  const [emailCaptchaId, setEmailCaptchaId] = useState('');
  const [emailCaptchaCode, setEmailCaptchaCode] = useState('');
  const [emailCaptchaImg, setEmailCaptchaImg] = useState('');
  const [emailCaptchaRequired, setEmailCaptchaRequired] = useState(false);
  const [emailCodeSent, setEmailCodeSent] = useState(false);
  const [emailCountdown, setEmailCountdown] = useState(0);

  // Mobile tab state
  const [mobile, setMobile] = useState('');
  const [mobileCode, setMobileCode] = useState('');
  const [mobileCaptchaId, setMobileCaptchaId] = useState('');
  const [mobileCaptchaCode, setMobileCaptchaCode] = useState('');
  const [mobileCaptchaImg, setMobileCaptchaImg] = useState('');
  const [mobileCaptchaRequired, setMobileCaptchaRequired] = useState(false);
  const [mobileCodeSent, setMobileCodeSent] = useState(false);
  const [mobileCountdown, setMobileCountdown] = useState(0);

  const [activeTab, setActiveTab] = useState('password');

  useEffect(() => {
    if (!allowedLoginMethods.includes(activeTab as LoginMethod)) {
      setActiveTab(allowedLoginMethods[0] ?? 'password');
    }
  }, [activeTab, allowedLoginMethods]);

  // Fetch captcha image
  const fetchCaptcha = useCallback(
    async (type: 'password' | 'email' | 'mobile') => {
      try {
        const res = await apiClient<CaptchaResponse>('/v1/auth/captcha');
        if (type === 'password') {
          setCaptchaId(res.id);
          setCaptchaImg(toDataUri(res.img));
        } else if (type === 'email') {
          setEmailCaptchaId(res.id);
          setEmailCaptchaImg(toDataUri(res.img));
        } else {
          setMobileCaptchaId(res.id);
          setMobileCaptchaImg(toDataUri(res.img));
        }
      } catch {
        toast.error(t('failedToLoadCaptcha'));
      }
    },
    [t]
  );

  // Send verification code
  const sendCode = useCallback(
    async (type: 'email' | 'mobile', value: string, captchaId: string, captchaCode: string) => {
      try {
        await apiClient('/v1/auth/code', {
          method: 'POST',
          body: JSON.stringify({
            codeType: type === 'email' ? 'EMAIL' : 'MOBILE',
            username: value,
            captchaId,
            captchaCode
          })
        });
        toast.success(t('verificationCodeSent'));
        if (type === 'email') {
          setEmailCodeSent(true);
          setEmailCountdown(60);
        } else {
          setMobileCodeSent(true);
          setMobileCountdown(60);
        }
      } catch (err) {
        // If backend returns captcha required error, show captcha
        if (err instanceof CastorApiError && err.code === ERR_CODE_INVALID_CAPTCHA) {
          if (type === 'email') {
            setEmailCaptchaRequired(true);
          } else {
            setMobileCaptchaRequired(true);
          }
          fetchCaptcha(type);
          toast.error(t('captchaRequired'));
        } else {
          toast.error(t('failedToSendCode'));
        }
      }
    },
    [fetchCaptcha, t]
  );

  // Countdown timer
  useEffect(() => {
    if (emailCountdown <= 0 && mobileCountdown <= 0) return;
    const timer = setInterval(() => {
      setEmailCountdown((prev) => Math.max(0, prev - 1));
      setMobileCountdown((prev) => Math.max(0, prev - 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [emailCountdown, mobileCountdown]);

  // Initial captcha load — no longer needed by default, loaded on demand
  // (password captcha will be fetched if backend requires it)

  // Handle login
  const handleLogin = async () => {
    clearError();

    try {
      if (activeTab === 'password') {
        if (!username || !password) {
          toast.error(`${t('enterUsername')} & ${t('enterPassword')}`);
          return;
        }
        await login(
          {
            username,
            credential: password,
            captchaId: captchaRequired ? captchaId || undefined : undefined,
            captchaCode: captchaRequired ? captchaCode || undefined : undefined
          },
          'password'
        );
      } else if (activeTab === 'email') {
        if (!email || !emailCode) {
          toast.error(t('enterEmailAndCode'));
          return;
        }
        await login(
          {
            username: email,
            credential: emailCode,
            captchaId: emailCaptchaRequired ? emailCaptchaId || undefined : undefined,
            captchaCode: emailCaptchaRequired ? emailCaptchaCode || undefined : undefined
          },
          'email'
        );
      } else {
        if (!mobile || !mobileCode) {
          toast.error(t('enterMobileAndCode'));
          return;
        }
        await login(
          {
            username: mobile,
            credential: mobileCode,
            captchaId: mobileCaptchaRequired ? mobileCaptchaId || undefined : undefined,
            captchaCode: mobileCaptchaRequired ? mobileCaptchaCode || undefined : undefined
          },
          'mobile'
        );
      }

      toast.success(t('signInSuccess'));
      router.push(redirectPath);
    } catch (err) {
      // If captcha required error on login, show captcha for that tab
      if (err instanceof CastorApiError && err.code === ERR_CODE_INVALID_CAPTCHA) {
        if (activeTab === 'password') {
          setCaptchaRequired(true);
          fetchCaptcha('password');
          setCaptchaCode('');
        } else if (activeTab === 'email') {
          setEmailCaptchaRequired(true);
          fetchCaptcha('email');
          setEmailCaptchaCode('');
        } else if (activeTab === 'mobile') {
          setMobileCaptchaRequired(true);
          fetchCaptcha('mobile');
          setMobileCaptchaCode('');
        }
      } else if (activeTab === 'password' && captchaRequired) {
        fetchCaptcha('password');
        setCaptchaCode('');
      }
    }
  };

  return (
    <div className='flex min-h-screen items-center justify-center bg-muted/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <CardTitle className='text-2xl'>{t('signInTitle')}</CardTitle>
          <CardDescription>{t('signInSubtitle')}</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList
              className='grid w-full'
              style={{
                gridTemplateColumns: `repeat(${allowedLoginMethods.length}, minmax(0, 1fr))`
              }}
            >
              {allowedLoginMethods.includes('password') && (
                <TabsTrigger value='password'>{tc('password')}</TabsTrigger>
              )}
              {allowedLoginMethods.includes('email') && (
                <TabsTrigger value='email'>
                  {t('email')} {tc('code', { defaultValue: 'Code' })}
                </TabsTrigger>
              )}
              {allowedLoginMethods.includes('mobile') && (
                <TabsTrigger value='mobile'>
                  {tc('mobile', { defaultValue: 'Mobile' })} {tc('code', { defaultValue: 'Code' })}
                </TabsTrigger>
              )}
            </TabsList>

            {/* Password Login */}
            <TabsContent value='password' className='mt-4 space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='username'>{t('username')}</Label>
                <Input
                  id='username'
                  type='text'
                  placeholder={t('enterUsername')}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete='username'
                  onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='password'>{t('password')}</Label>
                <PasswordInput
                  value={password}
                  onChange={setPassword}
                  placeholder={t('enterPassword')}
                  onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                />
              </div>
              {captchaRequired && captchaImg && (
                <div className='space-y-2'>
                  <Label htmlFor='captcha'>{t('captcha')}</Label>
                  <div className='flex gap-2'>
                    <Input
                      id='captcha'
                      type='text'
                      placeholder={t('captchaPlaceholder')}
                      value={captchaCode}
                      onChange={(e) => setCaptchaCode(e.target.value)}
                      className='flex-1'
                      maxLength={10}
                      onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                    />
                    <CaptchaImage
                      src={captchaImg}
                      onRefresh={() => fetchCaptcha('password')}
                      refreshTitle={t('clickToRefresh')}
                      refreshLabel={t('refreshCaptcha')}
                      alt={t('captchaAlt')}
                    />
                  </div>
                </div>
              )}
              <div className='flex items-center gap-2'>
                <input
                  type='checkbox'
                  id='remember'
                  checked={rememberMe}
                  onChange={(e) => setRememberMe(e.target.checked)}
                  className='h-4 w-4 rounded border-gray-300'
                />
                <Label htmlFor='remember' className='text-sm'>
                  {t('rememberMe')}
                </Label>
              </div>
            </TabsContent>

            {/* Email Code Login */}
            <TabsContent value='email' className='mt-4 space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='email'>{t('email')}</Label>
                <div className='flex gap-2'>
                  <Input
                    id='email'
                    type='email'
                    placeholder={t('enterEmail')}
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className='flex-1'
                    onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                  />
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={!email || (emailCodeSent && emailCountdown > 0)}
                    onClick={() => sendCode('email', email, emailCaptchaId, emailCaptchaCode)}
                  >
                    {emailCountdown > 0
                      ? `${emailCountdown}s`
                      : tc('send', { defaultValue: 'Send Code' })}
                  </Button>
                </div>
              </div>
              {emailCaptchaRequired && (
                <div className='space-y-2'>
                  <Label htmlFor='email-captcha'>{t('captcha')}</Label>
                  <div className='flex gap-2'>
                    <Input
                      id='email-captcha'
                      type='text'
                      placeholder={t('captchaPlaceholder')}
                      value={emailCaptchaCode}
                      onChange={(e) => setEmailCaptchaCode(e.target.value)}
                      className='flex-1'
                      maxLength={10}
                      onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                    />
                    <CaptchaImage
                      src={emailCaptchaImg}
                      onRefresh={() => fetchCaptcha('email')}
                      refreshTitle={t('clickToRefresh')}
                      refreshLabel={t('refreshCaptcha')}
                      alt={t('captchaAlt')}
                    />
                  </div>
                </div>
              )}
              <div className='space-y-2'>
                <Label htmlFor='email-code'>
                  {tc('verificationCode', { defaultValue: 'Verification Code' })}
                </Label>
                <Input
                  id='email-code'
                  type='text'
                  placeholder={tc('enterCodeFromEmail', { defaultValue: 'Enter code from email' })}
                  value={emailCode}
                  onChange={(e) => setEmailCode(e.target.value)}
                  maxLength={10}
                  onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                />
              </div>
            </TabsContent>

            {/* Mobile Code Login */}
            <TabsContent value='mobile' className='mt-4 space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='mobile'>
                  {tc('mobile', { defaultValue: 'Mobile' })}{' '}
                  {tc('number', { defaultValue: 'Number' })}
                </Label>
                <div className='flex gap-2'>
                  <Input
                    id='mobile'
                    type='tel'
                    placeholder={tc('enterMobileNumber', {
                      defaultValue: 'Enter your mobile number'
                    })}
                    value={mobile}
                    onChange={(e) => setMobile(e.target.value)}
                    className='flex-1'
                    onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                  />
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={!mobile || (mobileCodeSent && mobileCountdown > 0)}
                    onClick={() => sendCode('mobile', mobile, mobileCaptchaId, mobileCaptchaCode)}
                  >
                    {mobileCountdown > 0
                      ? `${mobileCountdown}s`
                      : tc('send', { defaultValue: 'Send Code' })}
                  </Button>
                </div>
              </div>
              {mobileCaptchaRequired && (
                <div className='space-y-2'>
                  <Label htmlFor='mobile-captcha'>{t('captcha')}</Label>
                  <div className='flex gap-2'>
                    <Input
                      id='mobile-captcha'
                      type='text'
                      placeholder={t('captchaPlaceholder')}
                      value={mobileCaptchaCode}
                      onChange={(e) => setMobileCaptchaCode(e.target.value)}
                      className='flex-1'
                      maxLength={10}
                      onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                    />
                    <CaptchaImage
                      src={mobileCaptchaImg}
                      onRefresh={() => fetchCaptcha('mobile')}
                      refreshTitle={t('clickToRefresh')}
                      refreshLabel={t('refreshCaptcha')}
                      alt={t('captchaAlt')}
                    />
                  </div>
                </div>
              )}
              <div className='space-y-2'>
                <Label htmlFor='mobile-code'>
                  {tc('verificationCode', { defaultValue: 'Verification Code' })}
                </Label>
                <Input
                  id='mobile-code'
                  type='text'
                  placeholder={tc('enterCodeFromSMS', { defaultValue: 'Enter code from SMS' })}
                  value={mobileCode}
                  onChange={(e) => setMobileCode(e.target.value)}
                  maxLength={10}
                  onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                />
              </div>
            </TabsContent>
          </Tabs>

          {error && (
            <div className='mt-4 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive'>
              {error}
            </div>
          )}

          <Button className='mt-4 w-full' size='lg' isLoading={isSubmitting} onClick={handleLogin}>
            {t('signIn')}
          </Button>

          <p className='mt-4 text-center text-sm text-muted-foreground'>
            {t('forgotPassword')}{' '}
            <Link
              href='/auth/reset-password'
              className='text-primary underline hover:text-primary/80'
            >
              {t('resetPassword')}
            </Link>
          </p>
          <p className='mt-2 text-center text-sm text-muted-foreground'>
            {t('dontHaveAccount')}{' '}
            <Link href='/auth/sign-up' className='text-primary underline hover:text-primary/80'>
              {t('signUpNow')}
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

// ============================================================
// Password Input with show/hide toggle
// ============================================================

function PasswordInput({
  value,
  onChange,
  onKeyDown,
  placeholder = 'Enter your password'
}: {
  value: string;
  onChange: (val: string) => void;
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  placeholder?: string;
}) {
  const [showPassword, setShowPassword] = useState(false);

  return (
    <div className='relative'>
      <Input
        type={showPassword ? 'text' : 'password'}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        autoComplete='current-password'
        onKeyDown={onKeyDown}
        className='pr-10'
      />
      <button
        type='button'
        className='absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground'
        onClick={() => setShowPassword(!showPassword)}
        tabIndex={-1}
      >
        {showPassword ? <Icons.eyeOff className='h-4 w-4' /> : <Icons.lock className='h-4 w-4' />}
      </button>
    </div>
  );
}
