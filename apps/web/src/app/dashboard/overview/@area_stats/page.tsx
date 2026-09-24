import { AreaGraph } from '@/features/overview/components/area-graph';
import { getMonthlyTrend, trendMonthLabel } from '@/features/dashboard/api/service';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { getLocale } from 'next-intl/server';

export default async function AreaStats() {
  const headers = await getServerAuthHeaders();
  const locale = await getLocale();
  const trend = await getMonthlyTrend({ headers }).catch(() => []);

  return (
    <AreaGraph
      data={trend.map((month) => ({
        month: trendMonthLabel(month.month, locale),
        users: month.newUsers
      }))}
    />
  );
}
