'use client';

import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { AlertModal } from '@/components/modal/alert-modal';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { usePermission } from '@/hooks/use-permission';
import { localizedText } from '@/lib/i18n-text';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { deleteOIDCProviderMutation } from '../api/mutations';
import { oidcProvidersQueryOptions } from '../api/queries';
import type { OIDCProvider } from '../api/types';
import { ProviderFormSheet } from './provider-form-sheet';

type Editing = { provider?: OIDCProvider } | null;

function CopyValue({ value }: { value: string }) {
  const t = useTranslations('sso');
  return (
    <span className='flex min-w-0 items-center gap-2'>
      <code className='bg-muted min-w-0 rounded px-1.5 py-0.5 font-mono text-xs break-all'>
        {value}
      </code>
      <Button
        variant='ghost'
        size='sm'
        className='h-7 shrink-0 px-2'
        aria-label={t('copy')}
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(value);
            toast.success(t('copied'));
          } catch {
            toast.error(t('copyFailed'));
          }
        }}
      >
        <Icons.page />
      </Button>
    </span>
  );
}

export function SSOListing() {
  const t = useTranslations('sso');
  const locale = useLocale();
  const { data: providers = [], isPending } = useQuery(oidcProvidersQueryOptions());
  const [editing, setEditing] = useState<Editing>(null);
  const [deleting, setDeleting] = useState<OIDCProvider | null>(null);
  const canCreate = usePermission('/api/v1/admin/oidc-providers:POST');
  const canUpdate = usePermission('/api/v1/admin/oidc-providers/:id:PUT');
  const canDelete = usePermission('/api/v1/admin/oidc-providers/:id:DELETE');
  const remove = useMutation(
    mergeMutationOptions(deleteOIDCProviderMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleted'));
        setDeleting(null);
      },
      onError: (error) => toast.error(error.message || t('messages.deleteFailed'))
    })
  );

  if (isPending) {
    return <div className='bg-muted h-48 animate-pulse rounded-lg' />;
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <p className='text-muted-foreground max-w-3xl text-sm'>{t('intro')}</p>
        {canCreate && (
          <Button onClick={() => setEditing({})}>
            <Icons.add /> {t('add')}
          </Button>
        )}
      </div>

      {providers.length === 0 ? (
        <p className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
          {t('empty')}
        </p>
      ) : (
        <div className='grid gap-4 lg:grid-cols-2'>
          {providers.map((provider) => (
            <Card key={provider.id}>
              <CardHeader>
                <CardTitle className='flex flex-wrap items-center gap-2'>
                  {localizedText(provider.name, locale)}
                  <Badge variant={provider.isEnabled ? 'success' : 'secondary'}>
                    {provider.isEnabled ? t('enabled') : t('disabled')}
                  </Badge>
                  {provider.autoRegister && <Badge variant='outline'>{t('autoRegister')}</Badge>}
                </CardTitle>
                <CardDescription className='font-mono text-xs'>{provider.code}</CardDescription>
                <CardAction className='flex gap-1'>
                  {canUpdate && (
                    <Button
                      variant='ghost'
                      size='sm'
                      aria-label={t('edit')}
                      onClick={() => setEditing({ provider })}
                    >
                      <Icons.edit />
                    </Button>
                  )}
                  {canDelete && (
                    <Button
                      variant='ghost'
                      size='sm'
                      aria-label={t('delete')}
                      onClick={() => setDeleting(provider)}
                    >
                      <Icons.trash />
                    </Button>
                  )}
                </CardAction>
              </CardHeader>
              <CardContent>
                <dl className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm'>
                  <dt className='text-muted-foreground'>{t('fields.issuer')}</dt>
                  <dd className='min-w-0 font-mono text-xs break-all'>{provider.issuer}</dd>
                  <dt className='text-muted-foreground'>{t('fields.clientId')}</dt>
                  <dd className='min-w-0 font-mono text-xs break-all'>{provider.clientId}</dd>
                  <dt className='text-muted-foreground'>{t('fields.scopes')}</dt>
                  <dd className='font-mono text-xs'>{provider.scopes.join(' ')}</dd>
                  <dt className='text-muted-foreground'>{t('fields.callbackUrl')}</dt>
                  <dd className='min-w-0'>
                    {provider.callbackUrl ? (
                      <CopyValue value={provider.callbackUrl} />
                    ) : (
                      <span className='text-destructive text-xs'>{t('publicUrlMissing')}</span>
                    )}
                  </dd>
                </dl>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {editing && (
        <ProviderFormSheet provider={editing.provider} onClose={() => setEditing(null)} />
      )}
      <AlertModal
        isOpen={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
        loading={remove.isPending}
      />
    </div>
  );
}
