import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations('privacyPolicy');
  return { title: t('title'), robots: { index: false } };
}

export default async function PrivacyPolicyPage() {
  const t = await getTranslations('privacyPolicy');
  return (
    <div className='min-h-screen px-4 py-12 sm:px-6 lg:px-8'>
      <div className='mx-auto max-w-3xl space-y-8'>
        <h1 className='text-foreground text-3xl font-bold'>{t('title')}</h1>
        {(['introduction', 'collection', 'auth', 'usage', 'deployment', 'contact'] as const).map(
          (section) => (
            <section key={section}>
              <h2 className='text-foreground mb-3 text-xl font-semibold'>{t(`${section}Title`)}</h2>
              <p className='text-muted-foreground text-base leading-relaxed'>{t(section)}</p>
            </section>
          )
        )}
        <a
          href='https://github.com/castorworks/castor'
          className='text-primary font-medium hover:underline'
        >
          {t('projectLink')}
        </a>
      </div>
    </div>
  );
}
