import type { MetadataRoute } from 'next';
import { getSiteOrigin } from '@/features/site/api/site-origin';
import { SITE_INDEXABLE_PATHS } from '@/features/site/constants';

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const origin = await getSiteOrigin();
  if (!origin) return [];
  return SITE_INDEXABLE_PATHS.map((path) => ({ url: `${origin}${path === '/' ? '' : path}` }));
}
