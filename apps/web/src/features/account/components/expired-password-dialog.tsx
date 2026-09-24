'use client';

import Image from 'next/image';
import { useCallback, useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { usePasswordPolicy } from '@/hooks/use-password-policy';
import { apiClient } from '@/lib/api-client';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { checkPassword } from '@/lib/password-policy';
import { publicSettingsQueryOptions } from '@/features/settings/api/queries';
import type { CaptchaResponse } from '@/stores/auth-store';
import { changeExpiredPasswordMutation } from '../api/mutations';

const CAPTCHA_SETTING_KEY = 'feature.captcha.enabled';

function toDataUri(img: string) {
  return img.startsWith('data:') ? img : `data:image/png;base64,${img}`;
}

interface ExpiredPasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  username: string;
  /** The password the user just signed in with; the backend verifies it again. */
  currentPassword: string;
  /** Called with the new password once it is saved, so the sign-in form can reuse it. */
  onChanged: (newPassword: string) => void;
}

/**
 * Shown when sign-in fails because the password has expired (the account is
 * otherwise fine). The user cannot sign in to reach the profile page, so the
 * password is replaced here by proving the current one.
 */
export function ExpiredPasswordDialog({
  open,
  onOpenChange,
  username,
  currentPassword,
  onChanged
}: ExpiredPasswordDialogProps) {
  const t = useTranslations('auth');
  const passwordPolicy = usePasswordPolicy();
  const { data: publicSettings } = useQuery(publicSettingsQueryOptions());
  const captchaEnabled = publicSettings?.[CAPTCHA_SETTING_KEY] !== false;

  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [captchaImg, setCaptchaImg] = useState('');

  const fetchCaptcha = useCallback(async () => {
    try {
      const res = await apiClient<CaptchaResponse>('/v1/auth/captcha');
      setCaptchaId(res.id);
      setCaptchaImg(toDataUri(res.img));
      setCaptchaCode('');
    } catch {
      toast.error(t('failedToLoadCaptcha'));
    }
  }, [t]);

  useEffect(() => {
    if (!open) return;
    setNewPassword('');
    setConfirmPassword('');
    if (captchaEnabled) void fetchCaptcha();
  }, [open, captchaEnabled, fetchCaptcha]);

  const mutation = useMutation({
    ...mergeMutationOptions(changeExpiredPasswordMutation, {
      onSuccess: () => {
        toast.success(t('expiredPasswordChanged'));
        onChanged(newPassword);
        onOpenChange(false);
      },
      onError: (error) => {
        toast.error(error.message || t('failedToChangeExpiredPassword'));
        if (captchaEnabled) void fetchCaptcha();
      }
    })
  });

  const submit = () => {
    const issue = checkPassword(newPassword, passwordPolicy.policy);
    if (issue) {
      toast.error(passwordPolicy.messages[issue]);
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error(t('passwordMismatch'));
      return;
    }
    mutation.mutate({
      username,
      currentPassword,
      newPassword,
      captchaId: captchaEnabled ? captchaId : undefined,
      captchaCode: captchaEnabled ? captchaCode : undefined
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{t('passwordExpiredTitle')}</DialogTitle>
          <DialogDescription>{t('passwordExpiredDescription', { username })}</DialogDescription>
        </DialogHeader>
        <div className='flex flex-col gap-4'>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='expired-new-password'>{t('newPassword')}</Label>
            <Input
              id='expired-new-password'
              type='password'
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              autoComplete='new-password'
            />
            <p className='text-muted-foreground text-xs'>{passwordPolicy.hint}</p>
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='expired-confirm-password'>{t('confirmPassword')}</Label>
            <Input
              id='expired-confirm-password'
              type='password'
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              autoComplete='new-password'
              onKeyDown={(event) => event.key === 'Enter' && submit()}
            />
          </div>
          {captchaEnabled && (
            <div className='flex flex-col gap-2'>
              <Label htmlFor='expired-captcha'>{t('captcha')}</Label>
              <div className='flex gap-2'>
                <Input
                  id='expired-captcha'
                  value={captchaCode}
                  onChange={(event) => setCaptchaCode(event.target.value)}
                  placeholder={t('captchaPlaceholder')}
                  maxLength={10}
                  className='flex-1'
                  onKeyDown={(event) => event.key === 'Enter' && submit()}
                />
                {captchaImg && (
                  <button
                    type='button'
                    className='h-9 overflow-hidden rounded-md border border-border bg-white p-0.5'
                    onClick={() => void fetchCaptcha()}
                    title={t('clickToRefresh')}
                    aria-label={t('refreshCaptcha')}
                  >
                    <Image
                      src={captchaImg}
                      alt={t('captchaAlt')}
                      width={96}
                      height={36}
                      unoptimized
                      className='h-8 w-23 rounded-[3px] object-cover'
                    />
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button isLoading={mutation.isPending} disabled={mutation.isPending} onClick={submit}>
            {t('changeExpiredPassword')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
