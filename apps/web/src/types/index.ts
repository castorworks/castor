import { Icons } from '@/components/icons';

export interface PermissionCheck {
  permission?: string;
  permissions?: readonly string[];
  plan?: string;
  feature?: string;
  requireOrg?: boolean;
}

export interface NavItem {
  title: string;
  /** i18n key for translated title, e.g. 'nav.dashboard'. When set, UI components should use this for display. */
  titleKey?: string;
  url: string;
  disabled?: boolean;
  external?: boolean;
  shortcut?: [string, string];
  icon?: keyof typeof Icons;
  label?: string;
  description?: string;
  isActive?: boolean;
  items?: NavItem[];
  access?: PermissionCheck;
}

export interface NavGroup {
  label: string;
  /** i18n key for translated group label, e.g. 'nav.systemManagement' */
  labelKey?: string;
  items: NavItem[];
}

export interface NavItemWithChildren extends NavItem {
  items: NavItemWithChildren[];
}

export interface NavItemWithOptionalChildren extends NavItem {
  items?: NavItemWithChildren[];
}

export interface FooterItem {
  title: string;
  items: {
    title: string;
    href: string;
    external?: boolean;
  }[];
}

export type MainNavItem = NavItemWithOptionalChildren;

export type SidebarNavItem = NavItemWithChildren;
