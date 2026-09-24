import { PieGraph } from '@/features/overview/components/pie-graph';
import { getAssetCategoryStats } from '@/features/dashboard/api/service';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function Stats() {
  const headers = await getServerAuthHeaders();
  const categoryStats = await getAssetCategoryStats({ headers }).catch(() => []);

  return <PieGraph data={categoryStats} />;
}
