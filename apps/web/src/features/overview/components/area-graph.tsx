'use client';

import { Area, AreaChart, CartesianGrid, XAxis } from 'recharts';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent
} from '@/components/ui/chart';
import { useTranslations } from 'next-intl';

interface AreaGraphProps {
  data: { month: string; users: number; logins: number }[];
}

export function AreaGraph({ data }: AreaGraphProps) {
  const t = useTranslations('dashboard.overview.charts');

  const chartConfig = {
    users: {
      label: t('newUsers'),
      color: 'var(--chart-1)'
    },
    logins: {
      label: t('logins'),
      color: 'var(--chart-2)'
    }
  } satisfies ChartConfig;

  if (data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('userGrowth')}</CardTitle>
          <CardDescription>{t('noDataAvailable')}</CardDescription>
        </CardHeader>
        <CardContent>
          <p className='text-muted-foreground py-12 text-center text-sm'>{t('userGrowthEmpty')}</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('userGrowth')}</CardTitle>
        <CardDescription>{t('userGrowthDescription')}</CardDescription>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig}>
          <AreaChart accessibilityLayer data={data}>
            <CartesianGrid vertical={false} strokeDasharray='3 3' />
            <XAxis
              dataKey='month'
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              tickFormatter={(value) => value.slice(0, 3)}
            />
            <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
            <defs>
              <DottedBackgroundPattern config={chartConfig} />
            </defs>
            <Area
              dataKey='users'
              type='natural'
              fill='url(#dotted-background-pattern-users)'
              fillOpacity={0.4}
              stroke='var(--chart-1)'
              stackId='a'
              strokeWidth={0.8}
            />
            <Area
              dataKey='logins'
              type='natural'
              fill='url(#dotted-background-pattern-logins)'
              fillOpacity={0.4}
              stroke='var(--chart-2)'
              stackId='a'
              strokeWidth={0.8}
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

const DottedBackgroundPattern = ({ config }: { config: ChartConfig }) => {
  const items = Object.fromEntries(
    Object.entries(config).map(([key, value]) => [key, value.color])
  );
  return (
    <>
      {Object.entries(items).map(([key, value]) => (
        <pattern
          key={key}
          id={`dotted-background-pattern-${key}`}
          x='0'
          y='0'
          width='7'
          height='7'
          patternUnits='userSpaceOnUse'
        >
          <circle cx='5' cy='5' r='1.5' fill={value} opacity={0.5}></circle>
        </pattern>
      ))}
    </>
  );
};
