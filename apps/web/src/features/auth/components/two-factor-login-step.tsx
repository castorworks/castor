'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { CastorApiError, CastorErrorCode } from '@/lib/api-client';
import { useAuthStore } from '@/stores/auth-store';

interface TwoFactorLoginStepProps {
  challenge: string;
  /** Signed in: continue like a normal sign-in. */
  onSuccess: () => void;
  /** Back to the first step (cancelled, or the challenge expired). */
  onRestart: (message?: string) => void;
}

/** Second sign-in step: a 6-digit authenticator code, or a recovery code. */
export function TwoFactorLoginStep({ challenge, onSuccess, onRestart }: TwoFactorLoginStepProps) {
  const t = useTranslations('auth.twoFactor');
  const { loginWithTOTP, isSubmitting, error, clearError } = useAuthStore();
  const [code, setCode] = useState('');
  const [useRecovery, setUseRecovery] = useState(false);

  const submit = async () => {
    if (!code.trim()) return;
    try {
      await loginWithTOTP(challenge, code);
      onSuccess();
    } catch (err) {
      if (err instanceof CastorApiError && err.errorCode === CastorErrorCode.MFAChallengeExpired) {
        onRestart(err.message);
        return;
      }
      setCode('');
    }
  };

  return (
    <div className='space-y-4'>
      <div className='space-y-1'>
        <h3 className='font-semibold'>{t('title')}</h3>
        <p className='text-muted-foreground text-sm'>
          {useRecovery ? t('recoveryDescription') : t('description')}
        </p>
      </div>
      <div className='space-y-2'>
        <Label htmlFor='two-factor-code'>{useRecovery ? t('recoveryCode') : t('code')}</Label>
        <Input
          id='two-factor-code'
          value={code}
          autoComplete='one-time-code'
          inputMode={useRecovery ? 'text' : 'numeric'}
          maxLength={useRecovery ? 32 : 6}
          placeholder={useRecovery ? 'xxxx-xxxx-xxxx' : '123456'}
          onChange={(e) =>
            setCode(useRecovery ? e.target.value : e.target.value.replace(/\D/g, ''))
          }
          onKeyDown={(e) => e.key === 'Enter' && submit()}
        />
      </div>
      {error && (
        <div className='bg-destructive/10 text-destructive rounded-md px-3 py-2 text-sm'>
          {error}
        </div>
      )}
      <Button
        className='w-full'
        size='lg'
        isLoading={isSubmitting}
        disabled={!code.trim()}
        onClick={submit}
      >
        {t('verify')}
      </Button>
      <div className='flex flex-wrap justify-between gap-2 text-sm'>
        <button
          type='button'
          className='text-primary hover:text-primary/80 underline'
          onClick={() => {
            setUseRecovery(!useRecovery);
            setCode('');
            clearError();
          }}
        >
          {useRecovery ? t('useAuthenticator') : t('useRecoveryCode')}
        </button>
        <button
          type='button'
          className='text-muted-foreground hover:text-foreground underline'
          onClick={() => {
            clearError();
            onRestart();
          }}
        >
          {t('back')}
        </button>
      </div>
    </div>
  );
}
