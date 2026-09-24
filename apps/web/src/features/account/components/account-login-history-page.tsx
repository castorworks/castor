'use client';

import { useSuspenseQuery } from '@tanstack/react-query';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table';
import { Icons } from '@/components/icons';
import { accountLoginHistoryQueryOptions } from '../api/user-login-history-queries';
import { useDict } from '@/hooks/use-dict';
import { useTranslations } from 'next-intl';
import { useFormat } from '@/hooks/use-format';

function formatUserAgent(ua: string | undefined, unknown: string, other: string): string {
  if (!ua) return unknown;
  if (ua.includes('Chrome')) return 'Chrome';
  if (ua.includes('Firefox')) return 'Firefox';
  if (ua.includes('Safari')) return 'Safari';
  if (ua.includes('Edge')) return 'Edge';
  return other;
}

export default function AccountLoginHistoryPage() {
  const t = useTranslations('loginHistory');
  const dict = useDict();
  const fmt = useFormat();
  const { data } = useSuspenseQuery(accountLoginHistoryQueryOptions({ page: 1, pageSize: 10 }));

  const items = data?.list ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('recentActivity')}</CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className='text-muted-foreground py-8 text-center text-sm'>{t('none')}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('columns.method')}</TableHead>
                <TableHead>{t('columns.result')}</TableHead>
                <TableHead>{t('columns.ipAddress')}</TableHead>
                <TableHead className='hidden md:table-cell'>{t('columns.userAgent')}</TableHead>
                <TableHead>{t('columns.time')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((record) => (
                <TableRow key={record.id}>
                  <TableCell className='font-medium'>
                    {dict.label('login_method', record.loginMethod)}
                  </TableCell>
                  <TableCell>
                    {record.success ? (
                      <Badge variant='success'>
                        <Icons.check className='mr-1 h-3 w-3' /> {t('success')}
                      </Badge>
                    ) : (
                      <Badge variant='destructive'>
                        <Icons.close className='mr-1 h-3 w-3' /> {t('failed')}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className='font-mono text-sm'>{record.ipAddr}</TableCell>
                  <TableCell className='hidden text-muted-foreground md:table-cell'>
                    {formatUserAgent(record.userAgent, t('unknown'), t('other'))}
                  </TableCell>
                  <TableCell className='text-muted-foreground text-sm'>
                    {fmt.dateTime(record.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
