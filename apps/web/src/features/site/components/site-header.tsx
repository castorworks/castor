import Link from 'next/link';
import { cookies } from 'next/headers';
import { getTranslations } from 'next-intl/server';
import { Button } from '@/components/ui/button';
import { LanguageSwitcher } from '@/components/language-switcher';
import { ThemeModeToggle } from '@/components/themes/theme-mode-toggle';
import { getSiteInfo } from '../api/site-info';
import { SITE_REPOSITORY_URL } from '../constants';
import { SiteBrand } from './site-brand';

export async function SiteHeader() {
  const [site, t, ta, cookieStore] = await Promise.all([
    getSiteInfo(),
    getTranslations('site'),
    getTranslations('auth'),
    cookies()
  ]);
  // Presence only picks the call to action. The session itself is validated by
  // the route guard when /dashboard is opened, which also clears a stale cookie.
  const hasSession = cookieStore.has('jwt');

  return (
    <header className='bg-background/80 sticky top-0 z-20 border-b backdrop-blur-md'>
      <div className='mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6'>
        <div className='flex min-w-0 items-center gap-6'>
          <SiteBrand name={site.name} nameClassName='hidden sm:inline' />
          <nav className='text-muted-foreground hidden items-center gap-6 text-sm md:flex'>
            <Link href='/about' className='hover:text-foreground transition-colors'>
              {t('about')}
            </Link>
            <a href={SITE_REPOSITORY_URL} className='hover:text-foreground transition-colors'>
              {t('source')}
            </a>
          </nav>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          <ThemeModeToggle />
          <LanguageSwitcher />
          {hasSession ? (
            <Button asChild size='sm'>
              <Link href='/dashboard'>{t('console')}</Link>
            </Button>
          ) : (
            <>
              <Button asChild size='sm' variant={site.registerEnabled ? 'ghost' : 'default'}>
                <Link href='/auth/sign-in'>{ta('signIn')}</Link>
              </Button>
              {site.registerEnabled && (
                <Button asChild size='sm' className='hidden sm:inline-flex'>
                  <Link href='/auth/sign-up'>{ta('signUp')}</Link>
                </Button>
              )}
            </>
          )}
        </div>
      </div>
    </header>
  );
}
