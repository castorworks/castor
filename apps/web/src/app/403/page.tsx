import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { getTranslations } from 'next-intl/server';

export async function generateMetadata() {
  const t = await getTranslations('common');
  return { title: t('forbiddenTitle') };
}

export default async function ForbiddenPage() {
  const t = await getTranslations('common');
  return (
    <main className='flex min-h-screen items-center justify-center bg-muted/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <div className='mx-auto mb-2 flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive'>
            <Icons.shield className='size-6' />
          </div>
          <CardTitle>{t('forbiddenTitle')}</CardTitle>
          <CardDescription>{t('accessDenied')}</CardDescription>
        </CardHeader>
        <CardContent className='flex justify-center'>
          <Button asChild>
            <Link href='/dashboard/overview'>{t('backToDashboard')}</Link>
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
