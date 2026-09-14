import { AreaGraph } from '@/features/overview/components/area-graph';
import { apiClient } from '@/lib/api-client';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { formatMonthYear } from '@/lib/format';
import { getLocale } from 'next-intl/server';

interface TimeRecord {
  createdAt: string;
}

export default async function AreaStats() {
  const headers = await getServerAuthHeaders();
  const locale = await getLocale();

  // Fetch users grouped by month
  const usersRes = await apiClient<{ list: TimeRecord[]; total: number }>(
    '/v1/admin/users?page=1&pageSize=200&order=created_at+desc',
    { headers }
  ).catch(() => ({ list: [] as TimeRecord[], total: 0 }) as const);

  // Fetch logins grouped by month
  const loginsRes = await apiClient<{ list: TimeRecord[]; total: number }>(
    '/v1/admin/login-histories?page=1&pageSize=200&order=created_at+desc',
    { headers }
  ).catch(() => ({ list: [] as TimeRecord[], total: 0 }));

  const userMonths = aggregateByMonth(usersRes.list ?? [], locale);
  const loginMonths = aggregateByMonth(loginsRes.list ?? [], locale);

  // Merge by month
  const monthMap = new Map<string, { users: number; logins: number }>();

  for (const m of userMonths) {
    if (!monthMap.has(m.month)) monthMap.set(m.month, { users: 0, logins: 0 });
    monthMap.get(m.month)!.users = m.count;
  }
  for (const m of loginMonths) {
    if (!monthMap.has(m.month)) monthMap.set(m.month, { users: 0, logins: 0 });
    monthMap.get(m.month)!.logins = m.count;
  }

  const data = Array.from(monthMap.entries())
    .slice(-6)
    .map(([month, counts]) => ({ month, ...counts }));

  return <AreaGraph data={data} />;
}

function aggregateByMonth(items: TimeRecord[], locale: string): { month: string; count: number }[] {
  const monthMap = new Map<string, number>();

  for (const item of items) {
    const date = new Date(item.createdAt);
    const label = formatMonthYear(date, { locale });
    monthMap.set(label, (monthMap.get(label) ?? 0) + 1);
  }

  return Array.from(monthMap.entries()).map(([month, count]) => ({
    month: month.split(' ')[0],
    count
  }));
}
