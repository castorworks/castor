'use client';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from '@/components/ui/breadcrumb';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';
import { Icons } from '@/components/icons';
import { Fragment } from 'react';

export function Breadcrumbs() {
  const items = useBreadcrumbs();
  if (items.length === 0) return null;

  return (
    <Breadcrumb>
      <BreadcrumbList className='flex-nowrap'>
        {items.map((item, index) => (
          <Fragment key={`${index}-${item.title}`}>
            {index !== items.length - 1 && (
              <BreadcrumbItem className='hidden whitespace-nowrap lg:inline-flex'>
                {item.link ? (
                  <BreadcrumbLink href={item.link}>{item.title}</BreadcrumbLink>
                ) : (
                  item.title
                )}
              </BreadcrumbItem>
            )}
            {index < items.length - 1 && (
              <BreadcrumbSeparator className='hidden lg:block'>
                <Icons.slash />
              </BreadcrumbSeparator>
            )}
            {index === items.length - 1 && (
              <BreadcrumbPage className='truncate whitespace-nowrap'>{item.title}</BreadcrumbPage>
            )}
          </Fragment>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
