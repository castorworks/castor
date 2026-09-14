import PageContainer from '@/components/layout/page-container';
import { Badge } from '@/components/ui/badge';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardAction,
  CardFooter
} from '@/components/ui/card';
import { Icons } from '@/components/icons';
import React from 'react';
import { getDashboardStats } from '@/features/dashboard/api/service';
import { getServerAuthHeaders, serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations, getLocale } from 'next-intl/server';
import { formatNumber } from '@/lib/format';

export default async function OverViewLayout({
  sales,
  pie_stats,
  bar_stats,
  area_stats
}: {
  sales: React.ReactNode;
  pie_stats: React.ReactNode;
  bar_stats: React.ReactNode;
  area_stats: React.ReactNode;
}) {
  const headers = await getServerAuthHeaders();
  const t = await getTranslations('dashboard.overview');
  const tc = await getTranslations('common');
  const locale = await getLocale();
  const canView = await serverHasPermission('/api/v1/admin/dashboard/stats:GET');
  const stats = await getDashboardStats({ headers }).catch(() => ({
    totalUsers: 0,
    totalAssets: 0,
    totalAssetSize: '0 B',
    todayLogins: 0,
    userGrowth: 0
  }));

  return (
    <PageContainer
      pageTitle={t('welcome')}
      access={canView}
      accessDeniedMessage={tc('accessDenied')}
    >
      <div className='flex flex-1 flex-col space-y-2'>
        <div className='*:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card dark:*:data-[slot=card]:bg-card grid grid-cols-1 gap-4 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:shadow-xs md:grid-cols-2 lg:grid-cols-4'>
          <Card className='@container/card'>
            <CardHeader>
              <CardDescription>{t('totalUsers')}</CardDescription>
              <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                {formatNumber(stats.totalUsers, { locale })}
              </CardTitle>
              <CardAction>
                <Badge variant='outline'>
                  <Icons.teams />
                  {t('managed')}
                </Badge>
              </CardAction>
            </CardHeader>
            <CardFooter className='flex-col items-start gap-1.5 text-sm'>
              <div className='line-clamp-1 flex gap-2 font-medium'>
                {t('systemUsers')} <Icons.teams className='size-4' />
              </div>
              <div className='text-muted-foreground'>{t('acrossAllRoles')}</div>
            </CardFooter>
          </Card>
          <Card className='@container/card'>
            <CardHeader>
              <CardDescription>{t('totalAssets')}</CardDescription>
              <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                {formatNumber(stats.totalAssets, { locale })}
              </CardTitle>
              <CardAction>
                <Badge variant='outline'>
                  <Icons.workspace />
                  {stats.totalAssetSize}
                </Badge>
              </CardAction>
            </CardHeader>
            <CardFooter className='flex-col items-start gap-1.5 text-sm'>
              <div className='line-clamp-1 flex gap-2 font-medium'>
                {t('uploadedFiles')} <Icons.workspace className='size-4' />
              </div>
              <div className='text-muted-foreground'>{t('totalStorageUsed')}</div>
            </CardFooter>
          </Card>
          <Card className='@container/card'>
            <CardHeader>
              <CardDescription>{t('todayLogins')}</CardDescription>
              <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                {formatNumber(stats.todayLogins, { locale })}
              </CardTitle>
              <CardAction>
                <Badge variant='outline'>
                  <Icons.login />
                  {t('activity')}
                </Badge>
              </CardAction>
            </CardHeader>
            <CardFooter className='flex-col items-start gap-1.5 text-sm'>
              <div className='line-clamp-1 flex gap-2 font-medium'>
                {t('loginRecords')} <Icons.login className='size-4' />
              </div>
              <div className='text-muted-foreground'>{t('todayLoginActivity')}</div>
            </CardFooter>
          </Card>
          <Card className='@container/card'>
            <CardHeader>
              <CardDescription>{t('systemHealth')}</CardDescription>
              <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                {t('online')}
              </CardTitle>
              <CardAction>
                <Badge variant='default' className='bg-green-600'>
                  <Icons.check />
                  {t('healthy')}
                </Badge>
              </CardAction>
            </CardHeader>
            <CardFooter className='flex-col items-start gap-1.5 text-sm'>
              <div className='line-clamp-1 flex gap-2 font-medium'>
                {t('allSystemsOperational')} <Icons.check className='size-4' />
              </div>
              <div className='text-muted-foreground'>{t('backendConnected')}</div>
            </CardFooter>
          </Card>
        </div>
        <div className='grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-7'>
          <div className='col-span-4'>{bar_stats}</div>
          <div className='col-span-4 md:col-span-3'>{sales}</div>
          <div className='col-span-4'>{area_stats}</div>
          <div className='col-span-4 min-h-0 md:col-span-3'>{pie_stats}</div>
        </div>
      </div>
    </PageContainer>
  );
}
