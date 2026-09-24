'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import type { AccountInfo, ContactType } from '../api/types';
import { ContactFormSheet } from './contact-form-sheet';

interface ContactSectionProps {
  accountInfo: AccountInfo;
}

export function ContactSection({ accountInfo }: ContactSectionProps) {
  const t = useTranslations('profile.contact');
  const [editing, setEditing] = useState<ContactType | null>(null);

  const rows = [
    {
      type: 'EMAIL' as const,
      label: t('email'),
      icon: Icons.mail,
      value: accountInfo.email,
      verified: accountInfo.emailVerified
    },
    {
      type: 'MOBILE' as const,
      label: t('mobile'),
      icon: Icons.phone,
      value: accountInfo.mobile,
      verified: accountInfo.mobileVerified
    }
  ];

  const hasVerifiedContact = rows.some((row) => row.value && row.verified);

  return (
    <Card className='gap-4'>
      <CardHeader className='pb-0'>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-3'>
        {rows.map((row) => (
          <div
            key={row.type}
            className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-border px-3 py-2.5'
          >
            <div className='flex min-w-0 items-center gap-3'>
              <row.icon className='text-muted-foreground shrink-0' />
              <div className='min-w-0'>
                <p className='text-muted-foreground text-xs'>{row.label}</p>
                <p className='truncate text-sm font-medium'>{row.value || t('notBound')}</p>
              </div>
            </div>
            <div className='flex items-center gap-2'>
              {row.value ? (
                <Badge variant={row.verified ? 'success' : 'warning'}>
                  {row.verified ? t('verified') : t('unverified')}
                </Badge>
              ) : (
                <Badge variant='outline'>{t('notBound')}</Badge>
              )}
              <Button size='sm' variant='outline' onClick={() => setEditing(row.type)}>
                {row.value ? t('change') : t('bind')}
              </Button>
            </div>
          </div>
        ))}
        {!hasVerifiedContact && (
          <p className='text-muted-foreground flex items-center gap-2 text-xs'>
            <Icons.warning className='text-warning shrink-0' />
            {t('recoveryHint')}
          </p>
        )}
      </CardContent>

      {editing && (
        <ContactFormSheet
          contactType={editing}
          currentContact={(editing === 'EMAIL' ? accountInfo.email : accountInfo.mobile) || ''}
          open
          onOpenChange={(open) => {
            if (!open) setEditing(null);
          }}
        />
      )}
    </Card>
  );
}
