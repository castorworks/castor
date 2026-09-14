'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import type { Role } from '../api/types';
import { getRoleUsers } from '../api/service';
import { roleKeys } from '../api/queries';

interface RoleUsersSheetProps {
  role: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RoleUsersSheet({ role, open, onOpenChange }: RoleUsersSheetProps) {
  const t = useTranslations('roles');
  const { data: users = [], isLoading } = useQuery({
    queryKey: roleKeys.users(role.code),
    queryFn: () => getRoleUsers(role.code),
    enabled: open
  });

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>
            {role.name} {t('card.users')}
          </SheetTitle>
          <SheetDescription className='font-mono'>{role.code}</SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto pr-1'>
          {isLoading ? (
            <div className='flex justify-center py-8'>
              <Icons.spinner className='h-6 w-6 animate-spin' />
            </div>
          ) : users.length > 0 ? (
            <div className='flex flex-wrap gap-2'>
              {users.map((username) => (
                <Badge key={username} variant='secondary' className='px-2 py-1'>
                  <Icons.user className='h-3 w-3' />
                  {username}
                </Badge>
              ))}
            </div>
          ) : (
            <p className='text-sm text-muted-foreground'>{t('users.noneAssigned')}</p>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
