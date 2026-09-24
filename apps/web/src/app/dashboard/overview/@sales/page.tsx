import { RecentSales } from '@/features/overview/components/recent-sales';
import { getRecentActivities } from '@/features/dashboard/api/service';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function Sales() {
  const headers = await getServerAuthHeaders();
  const activities = await getRecentActivities(5, { headers }).catch(() => []);

  return <RecentSales activities={activities} />;
}
