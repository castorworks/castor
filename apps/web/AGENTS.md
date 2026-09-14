# AGENTS.md — Castor Web

Next.js 16（App Router）管理后台，React 19 + TypeScript strict + shadcn/ui + TanStack Query/Form/Table + next-intl + Zustand，包管理器 Bun。
仓库级原则、验证命令、跨栈清单、安全不变量与前后端契约见 [../../AGENTS.md](../../AGENTS.md)，本文件只写前端细节。

## 1. 命令

```bash
bun install
bun run dev              # 开发服务器（后端默认 http://localhost:1234，可用 CASTOR_API_URL 覆盖）
bun run format           # oxfmt 写入；format:check 仅检查
bun run lint             # oxlint；lint:fix 自动修复
bunx tsc --noEmit        # 类型检查
bun run test             # vitest run；单文件：bunx vitest run <path>
bun run build
../../scripts/check.sh web   # 完整前端验证（与 CI 一致）
```

## 2. 目录

```
messages/{zh,en,ja,ko}.json   全部界面文案（key 集合必须一致）
src/
├── app/                      路由：/auth/*、/dashboard/*、/403、静态页；layout.tsx 挂 Provider 与 Toaster
├── features/{module}/        业务模块（见第 3 节）
├── components/
│   ├── ui/                   shadcn 基础组件、table/、tanstack-form.tsx —— 不直接修改
│   ├── layout/               侧边栏、头部、PageContainer、AuthProvider
│   ├── forms/ kbar/ modal/ themes/
│   ├── icons.tsx             Icons 对象（唯一图标入口）
│   └── permission-guard.tsx
├── config/nav-config.ts      把后端菜单树转换为侧边栏结构（无静态菜单）
├── hooks/                    use-data-table、use-permission、use-nav、use-dict 等
├── i18n/                     locales 配置、request.ts（cookie → Accept-Language → 默认 zh）
├── lib/                      api-client、server-auth-headers、query-client、mutation-utils、searchparams、
│                             castor-query、safe-redirect、security-headers、server-config
├── stores/auth-store.ts      当前用户、权限、登录/登出
├── styles/                   globals.css、theme.css、themes/*.css
└── proxy.ts                  /dashboard/* 路由守卫
```

现有模块：`account`、`assets`、`audit-logs`、`dashboard`、`dictionaries`、`login-histories`、`menus`、`notifications`、`overview`、`resources`、`roles`、`settings`、`users`。
**参考模板**：分页表格 + 表单抽屉抄 `features/resources`；类型安全表单字段参考 `features/users/components/user-form-sheet.tsx`。

## 3. 模块结构与写法

```
src/features/widgets/
├── api/types.ts             Widget、WidgetFilters、WidgetsPageResult、请求体类型
├── api/service.ts           调后端（唯一发请求的地方）
├── api/queries.ts           key factory + queryOptions
├── api/mutations.ts         mutationOptions（成功后 invalidate）
├── api/service.test.ts
├── schemas/widget.ts        zod schema 工厂 + 表单值类型
└── components/
    ├── widget-listing.tsx           服务端：预取 + HydrationBoundary
    ├── widget-form-sheet.tsx        客户端：Sheet + useAppForm，导出触发按钮
    └── widgets-table/{index,columns,cell-action}.tsx
src/app/dashboard/widgets/page.tsx
```

### service.ts

```ts
export async function getWidgets(
  filters: WidgetFilters,
  options?: RequestInit
): Promise<WidgetsPageResult> {
  const params = new URLSearchParams();
  params.set('page', String(filters.page ?? 1));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  const res = await apiClient<CastorListResponse<Widget>>(`/v1/admin/widgets?${params}`, options);
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? 1,
    pageSize: res.pageSize ?? 10,
    totalPages: res.totalPages ?? 0
  };
}
```

- `apiClient<T>(endpoint, options?)`（`@/lib/api-client`）：endpoint 不含 `/api` 前缀；自动解包 `{ code, data, message }`，非 200 抛 `CastorApiError(message, code, httpStatus)`；自动带 cookie、`Accept-Language`、`X-CSRF-Token`，401 时单次刷新重试。上传用 `apiUpload`。
- 函数最后一个参数接收 `options?: RequestInit`，供服务端组件传 `{ headers: await getServerAuthHeaders() }`。
- 排序用 `buildCastorOrder(sort, orderFields)`（`@/lib/castor-query`）生成 `field asc|desc`；筛选参数遵循后端 `{field}-{op}` 约定。
- 组件、hooks 不得直接 `fetch` 后端。

### queries.ts / mutations.ts

```ts
export const widgetKeys = {
  all: ['widgets'] as const,
  list: (filters: WidgetFilters) => [...widgetKeys.all, 'list', filters] as const,
  detail: (id: number) => [...widgetKeys.all, 'detail', id] as const
};
export const widgetsQueryOptions = (filters: WidgetFilters, options?: RequestInit) =>
  queryOptions({ queryKey: widgetKeys.list(filters), queryFn: () => getWidgets(filters, options) });

export const createWidgetMutation = mutationOptions({
  mutationFn: createWidget,
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: widgetKeys.all });
  }
});
```

组件中用 `useMutation({ ...mergeMutationOptions(createWidgetMutation, { onSuccess, onError }) })` 叠加 toast 等 UI 回调（基础回调先执行）。

### 页面

```tsx
export async function generateMetadata() {
  const t = await getTranslations();
  return { title: t('nav.widgets') };
}

export default async function WidgetsPage(props: { searchParams: Promise<SearchParams> }) {
  searchParamsCache.parse(await props.searchParams);
  const canView = await serverHasPermission('/api/v1/admin/widgets:GET');
  const canCreate = await serverHasPermission('/api/v1/admin/widgets:POST');
  const t = await getTranslations();
  return (
    <PageContainer
      pageTitle={t('nav.widgets')}
      pageDescription={t('dashboard.widgetsDescription')}
      access={canView}
      accessDeniedMessage={t('common.accessDenied')}
      pageHeaderAction={canCreate ? <WidgetFormSheetTrigger /> : undefined}
    >
      <WidgetListingPage />
    </PageContainer>
  );
}
```

- 页面标题/描述/操作按钮只通过 `PageContainer` props 传入。
- 列表服务端组件：从 `searchParamsCache` 读参数 → `try { await queryClient.prefetchQuery(...) } catch {}` → `<HydrationBoundary state={dehydrate(queryClient)}>`；预取失败时渲染空壳由客户端重试。
- 新的 URL 查询参数先加到 `src/lib/searchparams.ts`。
- 表格（`'use client'`）：`useQueryStates` + `getSortingStateParser(columnIds)` 读 URL 状态，`useSuspenseQuery` 取数据，`useDataTable({ data, columns, pageCount, shallow: true, debounceMs: 500 })`，渲染 `<DataTable><DataTableToolbar /></DataTable>`。
- 列：`DataTableColumnHeader` 表头；可筛选列设 `enableColumnFilter: true` 与 `meta: { label, placeholder, variant, icon }`；操作列 `cell-action.tsx` 用 `usePermission` 控制按钮。
- 表单：`useAppForm({ defaultValues, validators: { onSubmit: getWidgetSchema(messages) }, onSubmit })`，`<form.AppForm><form.Form id=...>` 内用 `form.TextField / SelectField / SwitchField`；需要类型安全字段名时用 `useFormFields<T>()`。不用 react-hook-form。
- zod 校验消息使用 i18n key，由组件翻译后传入 schema 工厂。

## 4. 导航与权限

- 侧边栏来自后端 `/api/v1/account/navigation`（`features/menus/api/queries.ts` → `hooks/use-nav.ts` → `config/nav-config.ts`）。菜单 `icon` 字符串必须是 `Icons` 的 key，否则回退为 `page`。
- `src/proxy.ts` 对 `/dashboard/*` 请求后端导航接口，按最长匹配的 `routes[].path` 与 `allowed` 放行，否则跳 `/403`；后端 401/403 清 cookie 跳登录，超时/5xx 返回 503。
- **新页面可见的前提**：后端 `DefaultResources` 登记资源，且存在对应菜单页节点（`apps/api/internal/infrastructure/bootstrap/menu.go` 或「菜单管理」）。前端没有静态路由表需要维护。
- 权限标识格式 `"{resourcePath}:{METHOD}"`：客户端 `usePermission / useAnyPermission / useAllPermissions`（`hooks/use-permission.ts`）、`PermissionGuard`；服务端 `serverHasPermission`。只做显隐，不做安全边界，不按角色名硬编码。

## 5. i18n

- 语言：`zh`（默认）、`en`、`ja`、`ko`；无 URL 语言段，语言存于 `NEXT_LOCALE` cookie。
- 服务端 `getTranslations()`，客户端 `useTranslations('namespace')`；后端种子名称（如 `seedResources.*`）用 `useSeedTranslation().ts()`。
- 新模块需要：`nav.{module}`、`dashboard.{module}Description`、模块命名空间（列、表单、消息、校验）、`seedResources.admin.{module}.*`、`resources.modules.{module}`。
- 后端错误的 `message` 已本地化，可直接 `toast.error(error.message || t('...'))`。
- 护栏：`src/i18n/messages.test.ts`（四语言 key 一致）、`src/features/resources/i18n.test.ts`（后端资源种子均有翻译）。

## 6. 约定

- 默认 Server Component；只有用到 hooks、浏览器 API、交互状态时加 `'use client'`。
- 图标只用 `@/components/icons` 的 `Icons`，不直接 import `@tabler/icons-react`。
- className 合并用 `cn()`；不修改 `src/components/ui/`，在业务组件中组合扩展。
- 状态：服务端数据用 TanStack Query；URL 状态用 nuqs；全局客户端状态用 Zustand（`stores/`）。
- 格式：oxfmt（单引号、JSX 单引号、无尾逗号、2 空格）；lint：oxlint（`.oxlintrc.json`）；路径别名 `@/*` → `src/*`。
- 主题：新增 `src/styles/themes/<name>.css`，在 `src/styles/theme.css` 引入，并在 `src/components/themes/theme.config.ts` 的 `THEMES` 注册。

## 7. 认证、配置与安全

- 登录态只存在于后端设置的 httpOnly `jwt` cookie，浏览器端不读取 token；服务端请求用 `getServerAuthHeaders()` 转发。
- 登录密码先用 `/v1/auth/public-key` 的 RSA 公钥加密。
- 任何 `redirect` 参数必须经 `safeRedirectPath`（`lib/safe-redirect.ts`）；站内链接解析用 `parseRelativePath`。
- 后端地址只能通过 `getCastorApiUrl()`（`lib/server-config.ts`）读取，仅限服务端；生产运行时缺少 `CASTOR_API_URL` 直接报错。`/api/v1` rewrite 在构建时固化。
- 安全响应头在 `lib/security-headers.ts`；引入新的外部脚本、样式、图片、字体或接口来源时同步更新 CSP 及其测试。
- `NEXT_PUBLIC_*` 会进入浏览器，只能放公开值；变量模板见 `.env.example`。

## 8. 测试

- Vitest（node 环境，无 DOM 测试库），测试与源码同目录 `*.test.ts(x)`。
- service 测试模式：

  ```ts
  vi.mock('@/lib/api-client', () => ({ apiClient: vi.fn() }));
  vi.mocked(apiClient).mockResolvedValueOnce({ total: 1, list: [...] });
  expect((await getWidgets({ page: 1 })).list).toHaveLength(1);
  ```

- 纯逻辑（`lib/`、schema、导航转换、proxy）必须有单测；安全相关助手修改时同步更新测试。
