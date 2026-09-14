import PageContainer from '@/components/layout/page-container';
import SettingsListingPage from '@/features/settings/components/settings-listing';
import { searchParamsCache } from '@/lib/searchparams';
import type { SearchParams } from 'nuqs/server';
import { SettingFormSheetTrigger } from '@/features/settings/components/settings-table/setting-form-sheet';
import { serverHasPermission } from '@/lib/server-auth-headers';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations();
  return {
    title: t('nav.settings')
  };
}

type PageProps = {
  searchParams: Promise<SearchParams>;
};

export default async function SettingsPage(props: PageProps) {
  const searchParams = await props.searchParams;
  searchParamsCache.parse(searchParams);
  const canViewSettings = await serverHasPermission('/api/v1/admin/settings:GET');
  const canManageSettings = await serverHasPermission('/api/v1/admin/settings:POST');
  const t = await getTranslations();

  return (
    <PageContainer
      pageTitle={t('nav.settings')}
      pageDescription={t('dashboard.settingsDescription')}
      access={canViewSettings}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canManageSettings ? <SettingFormSheetTrigger /> : undefined}
    >
      <SettingsListingPage />
    </PageContainer>
  );
}
