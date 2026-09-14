import PageContainer from '@/components/layout/page-container';
import DictionaryListingPage from '@/features/dictionaries/components/dictionary-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.dictionary')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function DictionariesPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewDictionaries = await serverHasPermission('/api/v1/admin/dict-types:GET');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.dictionary')}
      pageDescription={t('dashboard.dictionaryDescription')}
      access={canViewDictionaries}
      accessDeniedMessage={t('common.accessDenied')}
    >
      <DictionaryListingPage />
    </PageContainer>
  );
}
