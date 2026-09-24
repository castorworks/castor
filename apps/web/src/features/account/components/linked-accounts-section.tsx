'use client';

import { useEffect, useRef, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'sonner';
import { AlertModal } from '@/components/modal/alert-modal';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { localizedText } from '@/lib/i18n-text';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { unlinkIdentityMutation } from '@/features/sso/api/mutations';
import { myIdentitiesQueryOptions } from '@/features/sso/api/queries';
import { startIdentityLink } from '@/features/sso/api/service';
import type { UserIdentity } from '@/features/sso/api/types';

/** Sign-in providers the user can link to their account (shown only when any exist). */
export function LinkedAccountsSection() {
  const t = useTranslations('profile.identities');
  const ta = useTranslations('auth');
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { data: identities = [] } = useQuery(myIdentitiesQueryOptions());
  const [linking, setLinking] = useState<string | null>(null);
  const [unlinking, setUnlinking] = useState<UserIdentity | null>(null);
  const unlink = useMutation(
    mergeMutationOptions(unlinkIdentityMutation, {
      onSuccess: () => {
        toast.success(t('unlinked'));
        setUnlinking(null);
      },
      onError: (error) => toast.error(error.message)
    })
  );

  // The link callback returns here with ?linked=<code> or ?oidcError=<reason>: report once, then clean the URL.
  const reported = useRef(false);
  useEffect(() => {
    const linked = searchParams.get('linked');
    const error = searchParams.get('oidcError');
    if (reported.current || (!linked && !error)) return;
    reported.current = true;
    if (linked) toast.success(t('linked'));
    if (error)
      toast.error(
        ta.has(`oidcErrors.${error}`) ? ta(`oidcErrors.${error}`) : ta('oidcErrors.failed')
      );
    router.replace(pathname);
  }, [searchParams, router, pathname, t, ta]);

  const link = async (code: string) => {
    setLinking(code);
    try {
      window.location.assign(await startIdentityLink(code));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('linkFailed'));
      setLinking(null);
    }
  };

  if (identities.length === 0) return null;

  return (
    <Card className='gap-4'>
      <CardHeader className='pb-0'>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-3'>
        {identities.map((identity) => (
          <div
            key={identity.providerCode}
            className='border-border flex flex-wrap items-center justify-between gap-3 rounded-md border px-3 py-2.5'
          >
            <div className='flex min-w-0 items-center gap-3'>
              <Icons.shield className='text-muted-foreground shrink-0' />
              <div className='min-w-0'>
                <p className='flex items-center gap-2 text-sm font-medium'>
                  {localizedText(identity.providerName, locale)}
                  {identity.linked && <Badge variant='success'>{t('linkedBadge')}</Badge>}
                </p>
                {identity.linked && (
                  <p className='text-muted-foreground truncate text-xs' suppressHydrationWarning>
                    {identity.email || t('noEmail')}
                    {identity.lastLoginAt &&
                      ` · ${t('lastUsed', { time: formatDateTime(identity.lastLoginAt, { locale }) })}`}
                  </p>
                )}
              </div>
            </div>
            {identity.linked ? (
              <Button variant='outline' size='sm' onClick={() => setUnlinking(identity)}>
                {t('unlink')}
              </Button>
            ) : (
              identity.providerEnabled && (
                <Button
                  variant='outline'
                  size='sm'
                  isLoading={linking === identity.providerCode}
                  disabled={linking !== null}
                  onClick={() => link(identity.providerCode)}
                >
                  {t('link')}
                </Button>
              )
            )}
          </div>
        ))}
      </CardContent>
      <AlertModal
        isOpen={unlinking !== null}
        onClose={() => setUnlinking(null)}
        onConfirm={() => unlinking && unlink.mutate(unlinking.providerCode)}
        loading={unlink.isPending}
      />
    </Card>
  );
}
