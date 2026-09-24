'use client';

import Image from 'next/image';
import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { saveFile } from '@/lib/download';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import {
  beginTwoFactorSetupMutation,
  disableTwoFactorMutation,
  enableTwoFactorMutation,
  recoveryCodesFile,
  regenerateRecoveryCodesMutation,
  twoFactorStatusQueryOptions,
  type TwoFactorSetup
} from '../api/two-factor';

/** Fewer remaining recovery codes than this shows a warning. */
const LOW_RECOVERY_CODES = 3;

function CodeInput({
  id,
  value,
  onChange,
  onEnter,
  allowRecovery
}: {
  id: string;
  value: string;
  onChange: (value: string) => void;
  onEnter?: () => void;
  allowRecovery?: boolean;
}) {
  return (
    <Input
      id={id}
      value={value}
      autoComplete='one-time-code'
      inputMode={allowRecovery ? 'text' : 'numeric'}
      maxLength={allowRecovery ? 32 : 6}
      placeholder={allowRecovery ? '123456 / xxxx-xxxx-xxxx' : '123456'}
      onChange={(e) => onChange(allowRecovery ? e.target.value : e.target.value.replace(/\D/g, ''))}
      onKeyDown={(e) => e.key === 'Enter' && onEnter?.()}
    />
  );
}

/** Freshly generated recovery codes: shown once, with copy and download. */
function RecoveryCodesList({ codes }: { codes: string[] }) {
  const t = useTranslations('profile.twoFactor');
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(codes.join('\n'));
      toast.success(t('copied'));
    } catch {
      toast.error(t('copyFailed'));
    }
  };
  return (
    <div className='space-y-3'>
      <p className='text-sm'>{t('recoveryCodesHint')}</p>
      <ul className='bg-muted grid grid-cols-2 gap-x-6 gap-y-1 rounded-md p-4 font-mono text-sm'>
        {codes.map((code) => (
          <li key={code}>{code}</li>
        ))}
      </ul>
      <div className='flex flex-wrap gap-2'>
        <Button variant='outline' size='sm' onClick={copy}>
          <Icons.page /> {t('copy')}
        </Button>
        <Button
          variant='outline'
          size='sm'
          onClick={() =>
            saveFile({
              blob: new Blob([recoveryCodesFile(codes, window.location.host)], {
                type: 'text/plain'
              }),
              filename: 'recovery-codes.txt'
            })
          }
        >
          <Icons.download /> {t('download')}
        </Button>
      </div>
    </div>
  );
}

/** Setup: scan the QR code, confirm with a code, then save the recovery codes. */
function SetupDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useTranslations('profile.twoFactor');
  const tc = useTranslations('common');
  const [setup, setSetup] = useState<TwoFactorSetup | null>(null);
  const [code, setCode] = useState('');
  const [codes, setCodes] = useState<string[] | null>(null);
  const begin = useMutation({
    ...beginTwoFactorSetupMutation,
    onSuccess: setSetup,
    onError: (error) => toast.error(error.message)
  });
  const enable = useMutation(
    mergeMutationOptions(enableTwoFactorMutation, {
      onSuccess: (result) => {
        setCodes(result.recoveryCodes);
        toast.success(t('enabled'));
      },
      onError: (error) => {
        toast.error(error.message);
        setCode('');
      }
    })
  );
  // One setup request per opening; a failure is shown instead of retried in a loop.
  const [requested, setRequested] = useState(false);

  const close = () => {
    setSetup(null);
    setCode('');
    setCodes(null);
    setRequested(false);
    onClose();
  };

  // Each time the dialog opens, ask for a fresh secret (it replaces any unconfirmed one).
  const { mutate: beginSetup } = begin;
  useEffect(() => {
    if (open && !requested) {
      setRequested(true);
      beginSetup();
    }
  }, [open, requested, beginSetup]);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{codes ? t('saveRecoveryCodes') : t('setupTitle')}</DialogTitle>
          <DialogDescription>
            {codes ? t('saveRecoveryCodesDescription') : t('setupDescription')}
          </DialogDescription>
        </DialogHeader>
        {codes ? (
          <RecoveryCodesList codes={codes} />
        ) : setup ? (
          <div className='space-y-4'>
            <div className='flex flex-col items-center gap-3 sm:flex-row sm:items-start'>
              <Image
                src={setup.qrCode}
                width={160}
                height={160}
                unoptimized
                alt={t('qrAlt')}
                className='size-40 rounded-md border bg-white p-1'
              />
              <div className='min-w-0 space-y-1 text-sm'>
                <p className='text-muted-foreground'>{t('manualEntry')}</p>
                <code className='bg-muted block rounded px-2 py-1 font-mono text-xs break-all'>
                  {setup.secret}
                </code>
              </div>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='two-factor-setup-code'>{t('codeLabel')}</Label>
              <CodeInput
                id='two-factor-setup-code'
                value={code}
                onChange={setCode}
                onEnter={() => code.length === 6 && enable.mutate(code)}
              />
            </div>
          </div>
        ) : (
          <div className='bg-muted h-40 animate-pulse rounded-md' />
        )}
        <DialogFooter>
          {codes ? (
            <Button onClick={close}>{t('done')}</Button>
          ) : (
            <>
              <Button variant='outline' onClick={close}>
                {tc('cancel')}
              </Button>
              <Button
                isLoading={enable.isPending}
                disabled={!setup || code.length !== 6 || enable.isPending}
                onClick={() => enable.mutate(code)}
              >
                {t('enable')}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** Turn off (current password + code) or regenerate recovery codes (code). */
function ConfirmDialog({
  mode,
  onClose
}: {
  mode: 'disable' | 'regenerate' | null;
  onClose: () => void;
}) {
  const t = useTranslations('profile.twoFactor');
  const tc = useTranslations('common');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [codes, setCodes] = useState<string[] | null>(null);
  const close = () => {
    setPassword('');
    setCode('');
    setCodes(null);
    onClose();
  };
  const disable = useMutation(
    mergeMutationOptions(disableTwoFactorMutation, {
      onSuccess: () => {
        toast.success(t('disabled'));
        close();
      },
      onError: (error) => toast.error(error.message)
    })
  );
  const regenerate = useMutation(
    mergeMutationOptions(regenerateRecoveryCodesMutation, {
      onSuccess: (result) => setCodes(result.recoveryCodes),
      onError: (error) => toast.error(error.message)
    })
  );
  const pending = disable.isPending || regenerate.isPending;
  const submit = () => {
    if (!code.trim()) return;
    if (mode === 'disable') disable.mutate({ password, code });
    else regenerate.mutate(code);
  };

  return (
    <Dialog open={mode !== null} onOpenChange={(next) => !next && close()}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{mode === 'disable' ? t('disableTitle') : t('regenerateTitle')}</DialogTitle>
          <DialogDescription>
            {codes
              ? t('saveRecoveryCodesDescription')
              : mode === 'disable'
                ? t('disableDescription')
                : t('regenerateDescription')}
          </DialogDescription>
        </DialogHeader>
        {codes ? (
          <RecoveryCodesList codes={codes} />
        ) : (
          <div className='space-y-4'>
            {mode === 'disable' && (
              <div className='space-y-2'>
                <Label htmlFor='two-factor-password'>{t('currentPassword')}</Label>
                <Input
                  id='two-factor-password'
                  type='password'
                  autoComplete='current-password'
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
            )}
            <div className='space-y-2'>
              <Label htmlFor='two-factor-confirm-code'>{t('codeOrRecoveryLabel')}</Label>
              <CodeInput
                id='two-factor-confirm-code'
                value={code}
                onChange={setCode}
                onEnter={submit}
                allowRecovery
              />
            </div>
          </div>
        )}
        <DialogFooter>
          {codes ? (
            <Button onClick={close}>{t('done')}</Button>
          ) : (
            <>
              <Button variant='outline' onClick={close}>
                {tc('cancel')}
              </Button>
              <Button
                variant={mode === 'disable' ? 'destructive' : 'default'}
                isLoading={pending}
                disabled={!code.trim() || pending}
                onClick={submit}
              >
                {mode === 'disable' ? t('disable') : t('regenerate')}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function TwoFactorSection() {
  const t = useTranslations('profile.twoFactor');
  const locale = useLocale();
  const { data: status } = useQuery(twoFactorStatusQueryOptions());
  const [setupOpen, setSetupOpen] = useState(false);
  const [confirm, setConfirm] = useState<'disable' | 'regenerate' | null>(null);

  return (
    <Card className='gap-4'>
      <CardHeader className='pb-0'>
        <CardTitle className='flex flex-wrap items-center gap-2'>
          {t('title')}
          {status && (
            <Badge variant={status.enabled ? 'success' : 'secondary'}>
              {status.enabled ? t('on') : t('off')}
            </Badge>
          )}
        </CardTitle>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        {status?.enabled ? (
          <>
            <dl className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm'>
              {status.enabledAt && (
                <>
                  <dt className='text-muted-foreground'>{t('enabledSince')}</dt>
                  <dd suppressHydrationWarning>{formatDateTime(status.enabledAt, { locale })}</dd>
                </>
              )}
              <dt className='text-muted-foreground'>{t('recoveryCodesLeft')}</dt>
              <dd className='flex items-center gap-2'>
                {status.recoveryCodesRemaining}
                {status.recoveryCodesRemaining < LOW_RECOVERY_CODES && (
                  <Badge variant='warning'>{t('lowRecoveryCodes')}</Badge>
                )}
              </dd>
            </dl>
            <div className='flex flex-wrap gap-2'>
              <Button variant='outline' onClick={() => setConfirm('regenerate')}>
                <Icons.refresh /> {t('regenerate')}
              </Button>
              <Button variant='outline' onClick={() => setConfirm('disable')}>
                {t('disable')}
              </Button>
            </div>
          </>
        ) : (
          <Button onClick={() => setSetupOpen(true)} disabled={!status}>
            <Icons.shield /> {t('setup')}
          </Button>
        )}
      </CardContent>
      <SetupDialog open={setupOpen} onClose={() => setSetupOpen(false)} />
      <ConfirmDialog mode={confirm} onClose={() => setConfirm(null)} />
    </Card>
  );
}
