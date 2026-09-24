'use client';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarRail
} from '@/components/ui/sidebar';
import { useMediaQuery } from '@/hooks/use-media-query';
import { useNavigationGroups } from '@/hooks/use-nav';
import { useNavTranslations } from '@/hooks/use-nav-translations';
import { useTranslations } from 'next-intl';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { useAuthStore } from '@/stores/auth-store';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { assetUrl } from '@/lib/asset-url';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import * as React from 'react';
import { toast } from 'sonner';
import { Icons } from '../icons';
import type { NavItem } from '@/types';

export default function AppSidebar() {
  const pathname = usePathname();
  const { isOpen } = useMediaQuery();
  const filteredGroups = useNavigationGroups();
  const { getLabel } = useNavTranslations();
  const t = useTranslations();
  const { ts } = useSeedTranslation();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const setActiveRoles = useAuthStore((s) => s.setActiveRoles);

  const toggleActiveRole = async (roleCode: string, checked: boolean) => {
    if (!user) return;
    const current = user.activeRoles.map((role) => role.code);
    const next = checked ? [...current, roleCode] : current.filter((code) => code !== roleCode);
    try {
      await setActiveRoles(next);
      toast.success(t('layout.userNav.rolesUpdated'));
    } catch {
      toast.error(t('layout.userNav.rolesUpdateFailed'));
    }
  };

  React.useEffect(() => {
    // Side effects based on sidebar state changes
  }, [isOpen]);

  // Get user initials for avatar fallback
  const initials = user?.name
    ? user.name
        .split(' ')
        .map((n) => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2)
    : (user?.username?.slice(0, 2).toUpperCase() ?? 'U');

  return (
    <Sidebar collapsible='icon'>
      <SidebarHeader />
      <SidebarContent className='overflow-x-hidden'>
        {filteredGroups.map((group) => (
          <SidebarGroup key={group.label || 'ungrouped'} className='py-0'>
            {(group.label || group.labelKey) && (
              <SidebarGroupLabel>{getLabel(group)}</SidebarGroupLabel>
            )}
            <SidebarMenu>
              {group.items.map((item) => (
                <NavigationItem
                  key={item.url === '#' ? item.title : item.url}
                  item={item}
                  pathname={pathname}
                />
              ))}
            </SidebarMenu>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton
                  size='lg'
                  className='data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
                >
                  {user ? (
                    <>
                      <Avatar className='h-8 w-8'>
                        {user.avatar ? (
                          <AvatarImage src={assetUrl(user.avatar)} alt={user.name} />
                        ) : (
                          <AvatarFallback className='text-xs'>{initials}</AvatarFallback>
                        )}
                      </Avatar>
                      <div className='flex flex-1 flex-col truncate'>
                        <span className='truncate text-sm font-medium'>
                          {user.name || user.username}
                        </span>
                        <span className='truncate text-xs text-muted-foreground'>
                          {user.activeRoles.map((role) => ts(role.name)).join(', ') ||
                            t('layout.userNav.noActiveRole')}
                        </span>
                      </div>
                    </>
                  ) : (
                    <>
                      <Icons.user className='ml-auto size-4' />
                      <span className='truncate'>{t('layout.userNav.account')}</span>
                    </>
                  )}
                  <Icons.chevronsDown className='ml-auto size-4' />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                className='w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg'
                side='bottom'
                align='end'
                sideOffset={4}
              >
                {user ? (
                  <>
                    <DropdownMenuLabel className='p-0 font-normal'>
                      <div className='flex items-center gap-2 px-1 py-1.5'>
                        <Avatar className='h-8 w-8'>
                          {user.avatar ? (
                            <AvatarImage src={assetUrl(user.avatar)} alt={user.name} />
                          ) : (
                            <AvatarFallback className='text-xs'>{initials}</AvatarFallback>
                          )}
                        </Avatar>
                        <div className='flex flex-col'>
                          <span className='text-sm font-medium'>{user.name || user.username}</span>
                          <span className='text-xs text-muted-foreground'>@{user.username}</span>
                        </div>
                      </div>
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuLabel>{t('layout.userNav.activeRoles')}</DropdownMenuLabel>
                    {user.authorizedRoles.map((role) => (
                      <DropdownMenuCheckboxItem
                        key={role.code}
                        checked={user.activeRoles.some((active) => active.code === role.code)}
                        onSelect={(event) => event.preventDefault()}
                        onCheckedChange={(checked) => toggleActiveRole(role.code, checked === true)}
                      >
                        {ts(role.name)}
                      </DropdownMenuCheckboxItem>
                    ))}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem asChild>
                      <Link href='/dashboard/notifications'>
                        <Icons.notification className='mr-2 h-4 w-4' />
                        {t('layout.userNav.notifications')}
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => logout()}>
                      <Icons.logout className='mr-2 h-4 w-4' />
                      {t('layout.userNav.logOut')}
                    </DropdownMenuItem>
                  </>
                ) : (
                  <>
                    <DropdownMenuLabel className='p-0 font-normal'>
                      <div className='text-muted-foreground px-1 py-1.5 text-sm'>
                        {t('layout.userNav.signInToManage')}
                      </div>
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem asChild>
                      <Link href='/auth/sign-in'>
                        <Icons.login className='mr-2 h-4 w-4' />
                        {t('layout.userNav.signIn')}
                      </Link>
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function NavigationItem({ item, pathname }: { item: NavItem; pathname: string }) {
  const Icon = item.icon ? Icons[item.icon] : Icons.page;
  if (item.items?.length)
    return (
      <Collapsible
        asChild
        defaultOpen={item.items.some((child) => pathname.startsWith(child.url))}
        className='group/collapsible'
      >
        <SidebarMenuItem>
          <CollapsibleTrigger asChild>
            <SidebarMenuButton tooltip={item.title}>
              <Icon />
              <span>{item.title}</span>
              <Icons.chevronRight className='ml-auto' />
            </SidebarMenuButton>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <SidebarMenuSub>
              {item.items.map((child) => (
                <NavigationItem
                  key={child.url === '#' ? child.title : child.url}
                  item={child}
                  pathname={pathname}
                />
              ))}
            </SidebarMenuSub>
          </CollapsibleContent>
        </SidebarMenuItem>
      </Collapsible>
    );
  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild tooltip={item.title} isActive={pathname === item.url}>
        <Link href={item.url}>
          <Icon />
          <span>{item.title}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
