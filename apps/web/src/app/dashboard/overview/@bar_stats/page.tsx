import { BarGraph } from '@/features/overview/components/bar-graph';
import { apiClient } from '@/lib/api-client';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { formatMonthName } from '@/lib/format';
import { getLocale } from 'next-intl/server';

interface LoginRecord {
  createdAt: string;
}

export default async function BarStats() {
  const headers = await getServerAuthHeaders();
  const locale = await getLocale();

  // Fetch login histories and group by month
  const loginData = await apiClient<{ list: LoginRecord[]; total: number }>(
    '/v1/admin/login-histories?page=1&pageSize=100&order=created_at+desc',
    { headers }
  ).catch(() => ({ list: [] as LoginRecord[], total: 0 }));

  const monthlyLogins = aggregateByMonth(loginData.list ?? [], locale);

  return <BarGraph data={monthlyLogins} />;
}

function aggregateByMonth(
  items: LoginRecord[],
  locale: string
): { month: string; logins: number }[] {
  const monthMap = new Map<string, number>();

  for (const item of items) {
    const date = new Date(item.createdAt);
    // Use yyyy-MM as key for grouping (locale-independent)
    const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
    monthMap.set(key, (monthMap.get(key) ?? 0) + 1);
  }

  return Array.from(monthMap.entries())
    .slice(-6)
    .map(([key, count]) => {
      const [year, month] = key.split('-');
      const date = new Date(Number(year), Number(month) - 1);
      return { month: formatMonthName(date, { locale }), logins: count };
    });
}
