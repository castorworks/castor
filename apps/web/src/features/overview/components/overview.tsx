import PageContainer from '@/components/layout/page-container';
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  CardAction
} from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { AreaGraph } from './area-graph';
import { BarGraph } from './bar-graph';
import { PieGraph } from './pie-graph';
import { RecentSales } from './recent-sales';
import { Icons } from '@/components/icons';
import { Badge } from '@/components/ui/badge';

/**
 * @deprecated This component is no longer used.
 * The overview page now uses parallel routes in src/app/dashboard/overview/
 */
export default function OverViewPage() {
  return (
    <PageContainer>
      <div className='flex flex-1 flex-col space-y-2'>
        <div className='flex items-center justify-between space-y-2'>
          <h2 className='text-2xl font-bold tracking-tight'>Hi, Welcome back</h2>
        </div>
        <Tabs defaultValue='overview' className='space-y-4'>
          <TabsList>
            <TabsTrigger value='overview'>Overview</TabsTrigger>
            <TabsTrigger value='analytics' disabled>
              Analytics
            </TabsTrigger>
          </TabsList>
          <TabsContent value='overview' className='space-y-4'>
            <div className='*:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card dark:*:data-[slot=card]:bg-card grid grid-cols-1 gap-4 px-4 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:shadow-xs lg:px-6 @xl/main:grid-cols-2 @5xl/main:grid-cols-4'>
              <Card className='@container/card'>
                <CardHeader>
                  <CardDescription>Total Users</CardDescription>
                  <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                    0
                  </CardTitle>
                  <CardAction>
                    <Badge variant='outline'>
                      <Icons.teams />
                      Managed
                    </Badge>
                  </CardAction>
                </CardHeader>
                <CardFooter className='flex-col items-start gap-1.5 text-sm'>
                  <div className='line-clamp-1 flex gap-2 font-medium'>
                    System users <Icons.teams className='size-4' />
                  </div>
                  <div className='text-muted-foreground'>Across all roles</div>
                </CardFooter>
              </Card>
              <Card className='@container/card'>
                <CardHeader>
                  <CardDescription>Total Assets</CardDescription>
                  <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                    0
                  </CardTitle>
                  <CardAction>
                    <Badge variant='outline'>
                      <Icons.workspace />0 B
                    </Badge>
                  </CardAction>
                </CardHeader>
                <CardFooter className='flex-col items-start gap-1.5 text-sm'>
                  <div className='line-clamp-1 flex gap-2 font-medium'>
                    Uploaded files <Icons.workspace className='size-4' />
                  </div>
                  <div className='text-muted-foreground'>Total storage used</div>
                </CardFooter>
              </Card>
              <Card className='@container/card'>
                <CardHeader>
                  <CardDescription>Recent Logins</CardDescription>
                  <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                    0
                  </CardTitle>
                  <CardAction>
                    <Badge variant='outline'>
                      <Icons.login />
                      Activity
                    </Badge>
                  </CardAction>
                </CardHeader>
                <CardFooter className='flex-col items-start gap-1.5 text-sm'>
                  <div className='line-clamp-1 flex gap-2 font-medium'>
                    Login records <Icons.login className='size-4' />
                  </div>
                  <div className='text-muted-foreground'>All-time login history</div>
                </CardFooter>
              </Card>
              <Card className='@container/card'>
                <CardHeader>
                  <CardDescription>System Health</CardDescription>
                  <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                    Online
                  </CardTitle>
                  <CardAction>
                    <Badge variant='default' className='bg-green-600'>
                      <Icons.check />
                      Healthy
                    </Badge>
                  </CardAction>
                </CardHeader>
                <CardFooter className='flex-col items-start gap-1.5 text-sm'>
                  <div className='line-clamp-1 flex gap-2 font-medium'>
                    All systems operational <Icons.check className='size-4' />
                  </div>
                  <div className='text-muted-foreground'>Backend connected</div>
                </CardFooter>
              </Card>
            </div>
            <div className='grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-7'>
              <div className='col-span-4'>
                <BarGraph data={[]} />
              </div>
              <Card className='col-span-4 md:col-span-3'>
                <RecentSales activities={[]} />
              </Card>
              <div className='col-span-4'>
                <AreaGraph data={[]} />
              </div>
              <div className='col-span-4 md:col-span-3'>
                <PieGraph data={[]} />
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </PageContainer>
  );
}
