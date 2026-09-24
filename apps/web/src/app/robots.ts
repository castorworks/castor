import type { MetadataRoute } from 'next';
import { getSiteOrigin } from '@/features/site/api/site-origin';
import { SITE_PRIVATE_PATHS } from '@/features/site/constants';

export default async function robots(): Promise<MetadataRoute.Robots> {
  const origin = await getSiteOrigin();
  return {
    rules: { userAgent: '*', allow: '/', disallow: [...SITE_PRIVATE_PATHS] },
    ...(origin && { sitemap: `${origin}/sitemap.xml` })
  };
}
