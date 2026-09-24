import { BarGraph } from '@/features/overview/components/bar-graph';
import { getMonthlyTrend, trendMonthLabel } from '@/features/dashboard/api/service';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { getLocale } from 'next-intl/server';

export default async function BarStats() {
  const headers = await getServerAuthHeaders();
  const locale = await getLocale();
  const trend = await getMonthlyTrend({ headers }).catch(() => []);

  return (
    <BarGraph
      data={trend.map((month) => ({
        month: trendMonthLabel(month.month, locale),
        logins: month.logins
      }))}
    />
  );
}
