import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';

export default function ForbiddenPage() {
  return (
    <main className='flex min-h-screen items-center justify-center bg-muted/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <div className='mx-auto mb-2 flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive'>
            <Icons.shield className='size-6' />
          </div>
          <CardTitle>Access denied</CardTitle>
          <CardDescription>
            Your account does not have permission to open this page.
          </CardDescription>
        </CardHeader>
        <CardContent className='flex justify-center'>
          <Button asChild>
            <Link href='/dashboard/overview'>Back to dashboard</Link>
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
