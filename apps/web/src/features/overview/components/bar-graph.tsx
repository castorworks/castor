'use client';

import { Bar, BarChart, XAxis } from 'recharts';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent
} from '@/components/ui/chart';
import { useTranslations } from 'next-intl';

interface BarGraphProps {
  data: { month: string; logins: number }[];
}

export function BarGraph({ data }: BarGraphProps) {
  const t = useTranslations('dashboard.overview.charts');

  const chartConfig = {
    logins: {
      label: t('logins'),
      color: 'var(--chart-1)'
    }
  } satisfies ChartConfig;

  if (data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('monthlyActivity')}</CardTitle>
          <CardDescription>{t('noDataAvailable')}</CardDescription>
        </CardHeader>
        <CardContent>
          <p className='text-muted-foreground py-12 text-center text-sm'>
            {t('monthlyActivityEmpty')}
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('monthlyActivity')}</CardTitle>
        <CardDescription>{t('monthlyActivityDescription')}</CardDescription>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig}>
          <BarChart accessibilityLayer data={data}>
            <rect x='0' y='0' width='100%' height='85%' fill='url(#bar-graph-dots)' />
            <defs>
              <pattern
                id='bar-graph-dots'
                x='0'
                y='0'
                width='10'
                height='10'
                patternUnits='userSpaceOnUse'
              >
                <circle
                  className='dark:text-muted/40 text-muted'
                  cx='2'
                  cy='2'
                  r='1'
                  fill='currentColor'
                />
              </pattern>
            </defs>
            <XAxis
              dataKey='month'
              tickLine={false}
              tickMargin={10}
              axisLine={false}
              tickFormatter={(value) => value.slice(0, 3)}
            />
            <ChartTooltip
              cursor={false}
              content={<ChartTooltipContent indicator='dashed' hideLabel />}
            />
            <Bar dataKey='logins' fill='var(--chart-1)' radius={4} />
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}
