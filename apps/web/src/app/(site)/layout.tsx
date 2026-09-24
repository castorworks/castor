import type { Metadata } from 'next';
import { getSiteInfo } from '@/features/site/api/site-info';
import { SiteFooter } from '@/features/site/components/site-footer';
import { SiteHeader } from '@/features/site/components/site-header';

export async function generateMetadata(): Promise<Metadata> {
  const site = await getSiteInfo();
  return {
    title: { default: site.name, template: `%s · ${site.name}` },
    description: site.description,
    openGraph: {
      type: 'website',
      siteName: site.name,
      title: site.name,
      description: site.description
    }
  };
}

export default function SiteLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className='flex min-h-screen flex-col'>
      <SiteHeader />
      <main className='flex-1'>{children}</main>
      <SiteFooter />
    </div>
  );
}
