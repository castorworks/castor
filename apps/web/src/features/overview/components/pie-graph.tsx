'use client';

import { LabelList, Pie, PieChart } from 'recharts';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent
} from '@/components/ui/chart';
import type { AssetCategoryStat } from '@/features/dashboard/api/types';
import { useTranslations } from 'next-intl';

const COLORS = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)'
];

interface PieGraphProps {
  data: AssetCategoryStat[];
}

export function PieGraph({ data }: PieGraphProps) {
  const t = useTranslations('dashboard.overview.charts');

  const chartData = data.map((item, index) => ({
    category: item.category,
    count: item.count,
    fill: COLORS[index % COLORS.length]
  }));

  const chartConfig = data.reduce(
    (acc, item, index) => {
      acc[item.category] = {
        label: item.category.charAt(0).toUpperCase() + item.category.slice(1),
        color: COLORS[index % COLORS.length]
      };
      return acc;
    },
    {} as Record<string, { label: string; color: string }>
  );

  const config = {
    count: { label: t('count') },
    ...chartConfig
  } satisfies ChartConfig;

  if (chartData.length === 0) {
    return (
      <Card className='flex h-full flex-col'>
        <CardHeader className='items-center pb-0'>
          <CardTitle>{t('assetCategories')}</CardTitle>
          <CardDescription>{t('noAssetData')}</CardDescription>
        </CardHeader>
        <CardContent className='flex flex-1 items-center justify-center'>
          <p className='text-muted-foreground text-sm'>{t('noAssetsUploaded')}</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className='flex h-full flex-col'>
      <CardHeader className='items-center pb-0'>
        <CardTitle>{t('assetCategories')}</CardTitle>
        <CardDescription>{t('assetCategoriesDescription')}</CardDescription>
      </CardHeader>
      <CardContent className='flex flex-1 items-center justify-center pb-0'>
        <ChartContainer
          config={config}
          className='[&_.recharts-text]:fill-background mx-auto aspect-square max-h-[300px] min-h-[250px]'
        >
          <PieChart>
            <ChartTooltip content={<ChartTooltipContent nameKey='count' hideLabel />} />
            <Pie
              data={chartData}
              innerRadius={30}
              dataKey='count'
              nameKey='category'
              radius={10}
              cornerRadius={8}
              paddingAngle={4}
            >
              <LabelList
                dataKey='count'
                stroke='none'
                fontSize={12}
                fontWeight={500}
                fill='currentColor'
                formatter={(value: number) => value.toString()}
              />
            </Pie>
          </PieChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}
