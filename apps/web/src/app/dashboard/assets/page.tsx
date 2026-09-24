import PageContainer from '@/components/layout/page-container';
import AssetListingPage from '@/features/assets/components/asset-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { AssetUploadSheetTrigger } from '@/features/assets/components/asset-upload-sheet';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.assets')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function AssetsPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewAssets = await serverHasPermission('/api/v1/admin/assets:GET');
  const canManageAssets = await serverHasPermission('/api/v1/admin/assets:POST');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.assets')}
      pageDescription={t('dashboard.assetsDescription')}
      access={canViewAssets}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canManageAssets ? <AssetUploadSheetTrigger /> : undefined}
    >
      <AssetListingPage />
    </PageContainer>
  );
}
