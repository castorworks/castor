import Link from 'next/link';
import { cookies } from 'next/headers';
import { getTranslations } from 'next-intl/server';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { getSiteInfo } from '@/features/site/api/site-info';
import { SITE_REPOSITORY_URL } from '@/features/site/constants';

const FEATURES = [
  { key: 'rbac', icon: 'shield' },
  { key: 'menus', icon: 'panelLeft' },
  { key: 'assets', icon: 'workspace' },
  { key: 'audit', icon: 'clock' },
  { key: 'notifications', icon: 'notification' },
  { key: 'i18n', icon: 'globe' }
] as const;

// Product names, not copy: they read the same in every locale.
const STACK = ['Go', 'Gin', 'PostgreSQL', 'Redis', 'S3', 'Next.js', 'React', 'shadcn/ui'];

export default async function HomePage() {
  const [site, t, ts, ta, cookieStore] = await Promise.all([
    getSiteInfo(),
    getTranslations('home'),
    getTranslations('site'),
    getTranslations('auth'),
    cookies()
  ]);
  const primaryAction = cookieStore.has('jwt')
    ? { href: '/dashboard', label: ts('console') }
    : { href: '/auth/sign-in', label: ta('signIn') };

  return (
    <>
      <section className='relative overflow-hidden'>
        <div
          aria-hidden
          className='bg-primary/10 pointer-events-none absolute -top-40 left-1/2 size-160 -translate-x-1/2 rounded-full blur-3xl'
        />
        <div className='relative mx-auto flex max-w-4xl flex-col items-center px-4 py-20 text-center sm:px-6 sm:py-28'>
          <span className='bg-muted text-muted-foreground mb-6 inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium'>
            <Icons.sparkles className='text-primary size-3.5' />
            {site.name}
          </span>
          <h1 className='text-foreground text-4xl font-bold tracking-tight text-balance sm:text-5xl lg:text-6xl'>
            {t('heroTitle')}
          </h1>
          <p className='text-muted-foreground mt-6 max-w-2xl text-lg leading-relaxed text-pretty'>
            {t('heroSubtitle')}
          </p>
          <div className='mt-10 flex flex-col gap-3 sm:flex-row'>
            <Button asChild size='lg'>
              <Link href={primaryAction.href}>
                {primaryAction.label}
                <Icons.chevronRight />
              </Link>
            </Button>
            <Button asChild size='lg' variant='outline'>
              <a href={SITE_REPOSITORY_URL}>
                <Icons.github />
                {t('viewSource')}
              </a>
            </Button>
          </div>
        </div>
      </section>

      <section className='bg-muted/40 border-y'>
        <div className='mx-auto max-w-6xl px-4 py-20 sm:px-6'>
          <div className='mx-auto mb-12 max-w-2xl text-center'>
            <h2 className='text-foreground text-3xl font-bold tracking-tight'>
              {t('featuresTitle')}
            </h2>
            <p className='text-muted-foreground mt-4 text-lg'>{t('featuresSubtitle')}</p>
          </div>
          <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
            {FEATURES.map(({ key, icon }) => {
              const Icon = Icons[icon];
              return (
                <div key={key} className='bg-card rounded-xl border p-6 shadow-xs'>
                  <span className='bg-primary/10 text-primary mb-4 flex size-10 items-center justify-center rounded-lg'>
                    <Icon className='size-5' />
                  </span>
                  <h3 className='text-foreground font-semibold'>{t(`features.${key}.title`)}</h3>
                  <p className='text-muted-foreground mt-2 text-sm leading-relaxed'>
                    {t(`features.${key}.description`)}
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className='mx-auto max-w-6xl px-4 py-16 sm:px-6'>
        <p className='text-muted-foreground mb-6 text-center text-sm font-medium'>
          {t('stackTitle')}
        </p>
        <ul className='flex flex-wrap justify-center gap-2'>
          {STACK.map((name) => (
            <li
              key={name}
              className='bg-muted text-foreground rounded-full border px-3 py-1 text-sm'
            >
              {name}
            </li>
          ))}
        </ul>
      </section>

      <section className='mx-auto max-w-6xl px-4 pb-20 sm:px-6'>
        <div className='bg-primary text-primary-foreground flex flex-col items-center gap-6 rounded-2xl px-6 py-14 text-center'>
          <h2 className='text-3xl font-bold tracking-tight text-balance'>{t('ctaTitle')}</h2>
          <p className='text-primary-foreground/80 max-w-xl text-lg'>{t('ctaSubtitle')}</p>
          <Button asChild size='lg' variant='secondary'>
            <Link href={primaryAction.href}>{primaryAction.label}</Link>
          </Button>
        </div>
      </section>
    </>
  );
}
