import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { Icons } from '@/components/icons';
import { getSiteInfo } from '../api/site-info';
import { SITE_REPOSITORY_URL } from '../constants';
import { SiteBrand } from './site-brand';

export async function SiteFooter() {
  const [site, t] = await Promise.all([getSiteInfo(), getTranslations('site')]);
  const links = [
    { href: '/about', label: t('about') },
    { href: '/privacy-policy', label: t('privacy') },
    { href: '/terms-of-service', label: t('terms') }
  ];

  return (
    <footer className='border-t'>
      <div className='mx-auto flex max-w-6xl flex-col gap-8 px-4 py-10 sm:px-6 md:flex-row md:items-start md:justify-between'>
        <div className='max-w-sm space-y-3'>
          <SiteBrand name={site.name} />
          <p className='text-muted-foreground text-sm leading-relaxed'>{site.description}</p>
        </div>
        <nav className='text-muted-foreground flex flex-wrap items-center gap-x-6 gap-y-3 text-sm'>
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className='hover:text-foreground transition-colors'
            >
              {link.label}
            </Link>
          ))}
          <a
            href={SITE_REPOSITORY_URL}
            className='hover:text-foreground inline-flex items-center gap-1.5 transition-colors'
          >
            <Icons.github className='size-4' />
            GitHub
          </a>
        </nav>
      </div>
      <div className='text-muted-foreground mx-auto max-w-6xl px-4 pb-10 text-xs sm:px-6'>
        {t('copyright', { year: new Date().getFullYear(), name: site.name })}
      </div>
    </footer>
  );
}
