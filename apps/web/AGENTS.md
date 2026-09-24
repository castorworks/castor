# AGENTS.md — Castor Web

Next.js 16 (App Router) admin console: React 19 + TypeScript strict + shadcn/ui + TanStack Query/Form/Table + next-intl + Zustand; package manager Bun.
For repository-wide principles, verification commands, the cross-stack checklist, security invariants, and the frontend/backend contract, see [../../AGENTS.md](../../AGENTS.md). This file covers frontend details only.

## 1. Commands

```bash
bun install
bun run dev              # Dev server (backend defaults to http://localhost:1234; override with CASTOR_API_URL)
bun run format           # oxfmt write; format:check only checks
bun run lint             # oxlint; lint:fix auto-fixes
bunx tsc --noEmit        # Type check
bun run test             # vitest run; single file: bunx vitest run <path>
bun run build
python3 ../../scripts/check.py web   # Full frontend verification
```

## 2. Layout

```
messages/{zh,en,ja,ko}.json   All UI copy (key sets must match)
src/
├── app/                      Routes: (site)/ public site, /auth/*, /dashboard/*, /403, robots.ts, sitemap.ts; layout.tsx mounts Providers and Toaster
├── features/{module}/        Business modules (see section 3)
├── components/
│   ├── ui/                   shadcn base components, table/, tanstack-form.tsx — do not modify directly
│   ├── layout/               Sidebar, header, PageContainer, AuthProvider, DictProvider
│   ├── forms/ kbar/ modal/ themes/
│   ├── icons.tsx             Icons object (the only icon entry point)
│   ├── dict-badge.tsx        Dictionary item badge (label + color dot/icon)
│   └── permission-guard.tsx
├── config/nav-config.ts      Converts the backend menu tree into the sidebar structure (no static menu)
├── hooks/                    use-data-table, use-permission, use-nav, use-dict, etc.
├── i18n/                     Locales config, request.ts (cookie → Accept-Language → default zh)
├── lib/                      api-client, server-auth-headers, query-client, mutation-utils, searchparams,
│                             castor-query, safe-redirect, security-headers, server-config, dict, tag-color, asset-url
├── stores/auth-store.ts      Current user, permissions, login/logout
├── styles/                   globals.css, theme.css (token mapping), themes/*.css, design-tokens.test.ts
└── proxy.ts                  /dashboard/* route guard
```

Existing modules: `account`, `api-docs`, `assets`, `auth`, `audit-logs`, `dashboard`, `departments`, `dictionaries`, `jobs`, `login-histories`, `menus`, `notifications`, `overview`, `resources`, `roles`, `sessions`, `settings`, `site`, `sso`, `users`.
**Live notifications**: `NotificationCenter` (header) calls `useNotificationStream`, which opens an `EventSource` on `/api/v1/account/notifications/stream`, writes `unread` into the unread-count query and toasts `notification` events; on error it refreshes the token and reconnects with backoff (`reconnectDelay`). Admins see email delivery counts in the notifications table and per-recipient `emailStatus` (`notification_email_status` dictionary); the profile's `notification-preferences-section.tsx` toggles notification emails.
**Two-factor & SSO**: the sign-in page switches to `features/auth/components/two-factor-login-step.tsx` when login fails with `CastorErrorCode.TOTPRequired` (challenge in `CastorApiError.data`) or when the URL carries `mfaChallenge` (OIDC callback); it shows `oidcError` from the URL through `auth.oidcErrors.*` and lists enabled providers as plain links from `oidcAuthorizeUrl()` (a navigation, not a fetch). The profile has `two-factor-section.tsx` (setup with QR, recovery codes, disable/regenerate) and `linked-accounts-section.tsx` (link via the URL returned by `POST /account/identities/:provider/link`). Admins manage providers at `/dashboard/sso` (`features/sso`) and reset a user's two-factor from the users table.
**API docs**: `features/api-docs` renders the backend's OpenAPI document (`GET /api/v1/admin/openapi.json`, fetched raw via `apiDownload`) as a searchable, read-only list grouped by tag; `lib/schema.ts` resolves `$ref`s and formats types, and `api-docs-viewer.test.tsx` renders the committed `apps/api/openapi.json` so a generator change that breaks the viewer fails the web tests. There is no "try it" console by design (it would bypass CSRF and the UI's permission model).
**Export / import**: list services expose a `build*Query(filters)` shared by the list and `export*(filters, format)`. The export turns it into an export query with `exportQuery()` from `lib/download.ts`, which drops paging and adds `format` and the browser `tz`, and downloads it with `apiDownload()`. Tables render `<ExportButton onExport={(format) => export*(filters, format)} />` (`components/data-io/export-button.tsx`) in the toolbar when `usePermission('…/export:GET')`. `features/users/components/user-import-dialog.tsx` downloads the template, uploads with `apiUpload`, and shows `CastorApiError.data.errors` (per-row problems) when an import is rejected.
**Scheduled jobs**: `features/jobs` shows the jobs registered in backend code (cards: schedule, next and last run, run now, edit cron/enabled) and the run history table. Job names come from `jobs.catalog.{key}` via `useJobText()` (falls back to the key); both queries poll every 3 s while a run is `RUNNING`.
**Departments**: `GET /admin/departments` returns the whole tree flat, each node flagged `inScope` (inside the caller's data scope); build trees and indented options with `features/departments/tree.ts`, and only offer `inScope` departments for selection — the backend enforces the same rule.
**Reference templates**: for a paginated table + form sheet, copy `features/resources`; for type-safe form fields, see `features/users/components/user-form-sheet.tsx`.

## 3. Module structure and patterns

```
src/features/widgets/
├── api/types.ts             Widget, WidgetFilters, WidgetsPageResult, request body types
├── api/service.ts           Calls the backend (the only place that sends requests)
├── api/queries.ts           Key factory + queryOptions
├── api/mutations.ts         mutationOptions (invalidate on success)
├── api/service.test.ts
├── schemas/widget.ts        zod schema factory + form value types
└── components/
    ├── widget-listing.tsx           Server: prefetch + HydrationBoundary
    ├── widget-form-sheet.tsx        Client: Sheet + useAppForm, exports the trigger button
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

- `apiClient<T>(endpoint, options?)` (`@/lib/api-client`): the endpoint excludes the `/api` prefix; it unwraps `{ code, data, message }` automatically and throws `CastorApiError(message, code, httpStatus, errorCode)` on non-200 — branch on `errorCode` (`CastorErrorCode`), since `code` is only the HTTP status; it automatically sends cookies, `Accept-Language`, and `X-CSRF-Token`, and on 401 refreshes once and retries. Use `apiUpload` for uploads.
- Take `options?: RequestInit` as the function's last parameter so server components can pass `{ headers: await getServerAuthHeaders() }`.
- Build sorting with `buildCastorOrder(sort, orderFields)` (`@/lib/castor-query`), which produces `field asc|desc`; filter parameters follow the backend `{field}-{op}` convention.
- Components and hooks must not `fetch` the backend directly.

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

In components, use `useMutation({ ...mergeMutationOptions(createWidgetMutation, { onSuccess, onError }) })` to layer UI callbacks such as toasts on top (the base callbacks run first).

### Pages

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

- Pass the page title/description/action buttons only through `PageContainer` props.
- Listing server component: read params from `searchParamsCache` → `try { await queryClient.prefetchQuery(...) } catch {}` → `<HydrationBoundary state={dehydrate(queryClient)}>`; if prefetch fails, render the empty shell and let the client retry.
- Add new URL query parameters to this module's group in `src/lib/searchparams.ts` (parameters shared across modules go in `sharedParams`, noting which modules share them); `searchparams.test.ts` rejects cross-group name collisions and dead parameters nobody uses.
- Table (`'use client'`): read URL state with `useQueryStates` + `getSortingStateParser(columnIds)`, fetch data with `useSuspenseQuery`, call `useDataTable({ data: data.list, columns, rowCount: data.total, shallow: true, debounceMs: 500 })` (`rowCount` is the server's matching total; the page count and the footer's row count derive from it), and render `<DataTable><DataTableToolbar /></DataTable>`. `DataTable` fills its parent's height, so on a page where it sits below other content in normal flow give its wrapper a height (see `jobs-listing.tsx`). Page help panels (`infoContent`) are built from translations like `getUsersInfoContent(t)`; breadcrumbs come from the navigation menu's localized titles (`use-breadcrumbs.tsx`), so a new page needs no breadcrumb code.
- Columns: use `DataTableColumnHeader` for headers; for filterable columns set `enableColumnFilter: true` and `meta: { label, placeholder, variant, icon }`; the action column `cell-action.tsx` gates buttons with `usePermission`.
- Forms: `useAppForm({ defaultValues, validators: { onSubmit: getWidgetSchema(messages) }, onSubmit })`, and inside `<form.AppForm><form.Form id=...>` use `form.TextField / SelectField / SwitchField`; when you need type-safe field names, use `useFormFields<T>()`. Do not use react-hook-form.
- zod validation messages use i18n keys; the component translates them and passes them into the schema factory.
- Password fields: take `{ policy, messages, hint }` from `usePasswordPolicy()` (`@/hooks/use-password-policy`, reads the public password settings) and validate with `passwordSchema(policy, messages)` / `checkPassword` from `@/lib/password-policy`, which mirrors the backend rules; show `hint` as the field description. Never hard-code a minimum length.

## 4. Navigation and permissions

- The sidebar comes from the backend `/api/v1/account/navigation` (`features/menus/api/queries.ts` → `hooks/use-nav.ts` → `config/nav-config.ts`). A menu `icon` string must be a key of `Icons`; otherwise it falls back to `page`.
- `src/proxy.ts` calls the backend navigation endpoint for `/dashboard/*` requests and allows access by the longest-matching `routes[].path` and its `allowed`; otherwise it redirects to `/403`. On 401 (expired access token) it first refreshes with the cookies (`refreshSession`), forwards the rotated `Set-Cookie` to the browser and the rewritten `Cookie` to server components, and re-validates; if that refresh is rejected it redirects to login but keeps the cookie (another tab may just have rotated it). On 403 it clears the cookie and redirects to login; on timeout/5xx it returns 503.
- `/dashboard` is the console entry point (default redirect after login, and the public site's "Open console"): the guard falls through, in order, to the dashboard, the profile, and the first page the user has permission for.
- **Prerequisites for a new page to be visible**: the resource is registered in the backend `DefaultResources`, and a corresponding menu page node exists (`systemMenuTree` in `apps/api/internal/infrastructure/bootstrap/menu_tree.go`, or "Menu management"). The frontend has no static route table to maintain.
- Permission identifier format `"{resourcePath}:{METHOD}"`: on the client, `usePermission / useAnyPermission / useAllPermissions` (`hooks/use-permission.ts`) and `PermissionGuard`; on the server, `serverHasPermission`. Use them only for show/hide, never as a security boundary, and never hard-code role names.

### Public site

- The `src/app/(site)/` route group hosts pages that need no login: home `/`, `/about`, `/privacy-policy`, `/terms-of-service`, sharing the top bar and footer of `(site)/layout.tsx` (`features/site/components`). Put new public pages in this group and add them to `PUBLIC_PATHS` in `proxy.ts`; pages that search engines should index go into `SITE_INDEXABLE_PATHS` in `features/site/constants.ts` (included in `sitemap.xml`).
- Branding (site name, description) and the "Sign Up" button come from the public settings `site.name` / `site.description` / `feature.register.enabled`, read via `getSiteInfo()`; it uses the Next data cache (60 seconds), because all SSR requests hit the IP-rate-limited `/settings/public` from the same Web server IP. When the backend is unavailable it falls back to the default branding and public pages render as usual. Section copy lives in the `home` / `site` namespaces; to customize the home page, edit code and translations directly — do not build admin-editable section configuration.
- Public pages do not call the navigation endpoint or validate the session: the top bar decides between "Sign In" and "Open console" only by whether the `jwt` cookie exists; real validation happens on entering `/dashboard` (if invalid, it clears cookies and redirects to login).
- Projects that don't need a home page: replace `(site)/page.tsx` with `redirect('/dashboard')`.
- `robots.txt` / `sitemap.xml` are generated by `app/robots.ts` and `app/sitemap.ts`; absolute URLs take the Host forwarded by the ingress (the image does not bake in a domain). Do not add `public/robots.txt` again; it conflicts with them.

## 5. i18n

- Languages: `zh` (default), `en`, `ja`, `ko`; no URL locale segment — the locale is stored in the `NEXT_LOCALE` cookie.
- Server: `getTranslations()`; client: `useTranslations('namespace')`; backend seed names (e.g. `seedResources.*`) use `useSeedTranslation().ts()`.
- A new module needs: `nav.{module}`, `dashboard.{module}Description`, a module namespace (columns, form, messages, validation), `seedResources.admin.{module}.*`, `resources.modules.{module}`.
- Operator-managed data such as dictionary labels and menu titles are four-language objects (`I18nText`), not strings: render them with `localizedText(text, locale)` (`@/lib/i18n-text`); always read dictionary labels via `useDict()` / `<DictBadge>` (see "Dictionaries" in section 6).
- One sentence / one phrase is one message: do not concatenate several translations side by side (`{t('email')} {tc('code')}`, `` `${t('a')} & ${t('b')}` ``) — word order and spacing differ across languages; use ICU placeholders for counts, countdowns, and other variables (`{count, plural, …}`, `{seconds}`). Also do not write `t('key', { defaultValue })`; missing keys must be caught by tests, not shown as English. Put only truly generic words in `common` (`common.code` means an identifier code, not a verification code).
- Backend error `message` is already localized; use `toast.error(error.message || t('...'))` directly.

## 6. Conventions

- Default to Server Components; add `'use client'` only when using hooks, browser APIs, or interactive state.
- Use only `Icons` from `@/components/icons` for icons; do not import `@tabler/icons-react` directly.
- Merge classNames with `cn()`; do not modify `src/components/ui/` for business needs — compose and extend in business components (token-level variants are the exception, see below).
- Dictionaries: labels, colors, and icons for backend enum values (statuses, types, levels, audit types, etc.) come from the dictionary; do not write translation keys or color `switch`es for them.
  - For badges use `<DictBadge type='asset_status' value={row.original.status} />` (`@/components/dict-badge`); elsewhere use `const dict = useDict()` (`@/hooks/use-dict`): `dict.label(type, value)`, `dict.options(type)` (table filters / dropdowns), `dict.color()`, `dict.icon()`. Column factories are themselves hooks (`useXxxColumns`), so call `useDict()` inside and use it via closure. For form dropdowns use `dictOptionsOr(dict, type, fallback)`, which still works when the dictionary is empty.
  - The `type` parameter is typed `DictTypeCode`, with values from `DICT_TYPES` in `src/lib/dict.ts`, which maps one-to-one to the backend `DefaultDictTypes` (enforced by `lib/dict.test.ts`). To use a new dictionary in code, first register the seed in the backend, then add it to `DICT_TYPES`; types operators create ad hoc in the UI cannot be referenced by code.
  - Data is prefetched on the server by `app/dashboard/layout.tsx` and provided via `<DictProvider>`, so the first-paint HTML contains labels rather than raw values like `ACTIVE`, and after operators edit a dictionary every consumer across the site refreshes automatically. `useDict()` throws outside `<DictProvider>` — pages outside `/dashboard` that need dictionaries must mount the Provider themselves and use the unauthenticated public dictionary endpoint instead. **Do not read the query cache directly**: on the server `getQueryClient()` returns a brand-new empty instance every time, which renders raw values and causes hydration mismatches.
  - `label()` falls back to the raw value when the dictionary item is not found: disabled or historically existing values still appear in old data, so this fallback must not be changed to throw.
- File fields: files in business records are always the asset's `objectKey` (string), not a URL.
  - In forms use `<FormAssetField name='cover' label=… category='IMAGE' isPublic />` (`@/components/forms/fields`): it supports uploading on the spot (as a business attachment with `scope=ATTACHMENT`, reclaimed along with its references, not added to the asset library) or picking from the asset library, and the field value is the `objectKey`; when you need type-safe field names use `typedField<FormValues>()(FormAssetField)` (see the avatar field in `features/users/components/user-form-sheet.tsx`). The frontend only obtains the `objectKey`; reference registration is done by the backend business service.
  - For multi-file fields use `<FormAssetListField uploadPath=… uploadPermission=… max=… />`: the field value is `AssetAttachment[]`, and on submit take `{ objectKey, name }` from each item; `uploadPath` is the business module's own upload route (its permission decides who can upload). For read-only display use `features/assets/components/asset-attachment-list.tsx`; the download URL is provided by the business module (e.g. `notificationAttachmentUrl(id, key)`) — private attachments do not go through `assetUrl()`. See `features/notifications`.
  - For display use `assetUrl(objectKey)` (`@/lib/asset-url`): by default it is the unauthenticated public URL, accessible only for public + active + non-high-risk-type assets (avatars, covers); admin pages that download or preview private files use `assetUrl(objectKey, { admin: true })`. Do not hand-build `/api/v1/assets/...`.
  - For admin thumbnails use `features/assets/components/asset-thumbnail.tsx`, not `next/image`: its optimizer fetches images on the server without the user's cookie.
  - The backend rejects deleting/taking offline a referenced asset with 409; the error message is already localized, so use `toast.error(error.message || …)` directly. New business modules must register four-language names in `assets.references.ownerTypes.{ownerType}` and `assets.references.fields.{ownerType}.{field}` for the asset detail's "Used by" to show readable text.
- State: TanStack Query for server data; nuqs for URL state; Zustand (`stores/`) for global client state.
- Formatting: oxfmt (single quotes, JSX single quotes, no trailing commas, 2 spaces); lint: oxlint (`.oxlintrc.json`); path alias `@/*` → `src/*`.
- Design tokens: all style values go through tokens, defined in `src/styles/` and enforced by `src/styles/design-tokens.test.ts`.
  - For colors use semantic token utility classes (`bg-primary`, `text-muted-foreground`, `border-destructive`, etc.); Tailwind palette classes (`bg-green-600`, `text-zinc-500`, etc.) are forbidden.
  - Status colors: `success` / `warning` / `info` / `destructive` (each with `-foreground`); for badges use `<Badge variant='success|warning|info|destructive'>`.
  - Data-driven categorical colors (dictionary item colors, file types): `tag-{red,orange,yellow,green,blue,purple,pink,gray}`; the name-to-class mapping lives only in `tagColorBg()` in `@/lib/tag-color`; this list (`TAG_COLORS`) matches the backend `dictionary.Colors`. Present categorical colors as a "neutral badge + color dot" (`<DictBadge>`); do not use them as badge backgrounds: across all themes and dark mode, no single fixed foreground color guarantees contrast for all eight colors.
  - For font sizes use only type-scale utility classes (`text-2xs` 10px, `text-xs`, `text-sm` …); arbitrary values like `text-[11px]` are forbidden.
  - Controls that show translated text on one line (`SelectTrigger`, `SelectContent`, `DropdownMenuContent`, `TabsTrigger`, …) size to their content: no fixed `w-*`, which clips longer locales ("Englis…"); use `min-w-*` for a floor. Popovers that list translated labels likewise use `w-auto min-w-*`.
  - Color variables are oklch values; in arbitrary values write `var(--token)` directly — do not wrap it in `hsl()` / `rgb()`.
  - The token-to-Tailwind `@theme inline` mapping is defined only once, in `src/styles/theme.css`; theme-independent tokens (`--tag-*`, `--text-*`) also go in that file.
  - New color token: add it to both the light and dark blocks of every theme file, and map it in the `@theme inline` of `theme.css`. New font-size token: write it into `@theme inline` (`--text-<name>` + `--text-<name>--line-height`) and register it in `extendTailwindMerge` in `src/lib/utils.ts`; otherwise `cn()` merges it as a text color.
  - Modify `src/components/ui/` only when implementing token-level variants (e.g. Badge status variants).
- Themes: add `src/styles/themes/<name>.css` (containing only the two variable blocks `[data-theme='<name>']` and `.dark`, with variable names identical to other themes), import it in `src/styles/theme.css`, and register it in `THEMES` in `src/components/themes/theme.config.ts`. The file for `DEFAULT_THEME` additionally holds the `:root:not([data-theme])` fallback selector; move it along when changing the default theme.

## 7. Auth, configuration and security

- For the rules on login state, `redirect`, and CSP, see section 6 of the root AGENTS.md. Frontend touchpoints: server-side requests forward cookies with `getServerAuthHeaders()`; login passwords are first encrypted with the RSA public key from `/v1/auth/public-key`; in-site links are parsed with `parseRelativePath`.
- Read the backend URL only via `getCastorApiUrl()` (`lib/server-config.ts`), server-side only; a missing `CASTOR_API_URL` at production runtime raises an error immediately. The `/api/v1` rewrite is fixed at build time.
- When changing CSP, update the tests for `lib/security-headers.ts` accordingly.
- `NEXT_PUBLIC_*` reaches the browser; put only public values there; see `.env.example` for the variable template.

## 8. Testing

- Vitest (node environment, no DOM testing library); tests live next to the source as `*.test.ts(x)`.
- Service test pattern:

  ```ts
  vi.mock('@/lib/api-client', () => ({ apiClient: vi.fn() }));
  vi.mocked(apiClient).mockResolvedValueOnce({ total: 1, list: [...] });
  expect((await getWidgets({ page: 1 })).list).toHaveLength(1);
  ```

- Pure logic (`lib/`, schemas, navigation conversion, proxy) must have unit tests; when modifying security-related helpers, update their tests accordingly.
- Style guardrail: `src/styles/design-tokens.test.ts` (theme registration/files/imports consistent, variable names consistent across themes, every mapped token defined and with a dark value; no palette classes, no arbitrary font sizes, no `hsl(var(--x))` in source; no fixed widths on single-line text controls; font-size tokens registered with tailwind-merge). When it fails, fix the code or add the missing tokens — do not loosen the test.
