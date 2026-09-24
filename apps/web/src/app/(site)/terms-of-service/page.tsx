import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations('termsOfService');
  return { title: t('title'), robots: { index: false } };
}

const SECTIONS = [
  'introduction',
  'selfHosted',
  'openSource',
  'warranty',
  'data',
  'changes'
] as const;

export default async function TermsOfServicePage() {
  const t = await getTranslations('termsOfService');
  return (
    <div className='px-4 py-16 sm:px-6'>
      <div className='mx-auto max-w-3xl space-y-8'>
        <h1 className='text-foreground text-3xl font-bold'>{t('title')}</h1>
        {SECTIONS.map((section) => (
          <section key={section}>
            <h2 className='text-foreground mb-3 text-xl font-semibold'>{t(`${section}Title`)}</h2>
            <p className='text-muted-foreground text-base leading-relaxed'>{t(section)}</p>
          </section>
        ))}
        <p className='text-muted-foreground border-t pt-4 text-center text-sm'>{t('contact')}</p>
      </div>
    </div>
  );
}
