import { Card, CardHeader, CardContent, CardTitle, CardDescription } from '@/components/ui/card';
import type { DashboardRecentActivity } from '@/features/dashboard/api/types';
import { DictBadge } from '@/components/dict-badge';
import { formatTime } from '@/lib/format';
import { getLocale, getTranslations } from 'next-intl/server';

interface RecentSalesProps {
  activities: DashboardRecentActivity[];
}

export async function RecentSales({ activities }: RecentSalesProps) {
  const locale = await getLocale();
  const t = await getTranslations('dashboard.overview.charts');

  if (activities.length === 0) {
    return (
      <Card className='h-full'>
        <CardHeader>
          <CardTitle>{t('recentActivity')}</CardTitle>
          <CardDescription>{t('noActivityRecorded')}</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card className='h-full'>
      <CardHeader>
        <CardTitle>{t('recentActivity')}</CardTitle>
        <CardDescription>{t('recentActivityDescription')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className='space-y-6'>
          {activities.map((activity) => (
            <div key={activity.id} className='flex items-start gap-3'>
              <div className='mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted'>
                <span className='text-xs font-medium'>
                  {(activity.username ?? '?').charAt(0).toUpperCase()}
                </span>
              </div>
              <div className='min-w-0 flex-1'>
                <p className='text-sm leading-none font-medium'>
                  {activity.username ?? t('unknown')}
                </p>
                <p className='text-muted-foreground text-xs'>
                  {activity.action}
                  {activity.module && (
                    <DictBadge type='audit_log_type' value={activity.module} className='ml-1' />
                  )}
                </p>
              </div>
              <span className='text-muted-foreground/60 whitespace-nowrap text-xs'>
                {formatTime(new Date(activity.createdAt), { locale })}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
