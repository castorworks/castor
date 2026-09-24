/**
 * 资产下载地址。业务记录只存资产的 objectKey，展示时经这里换成 URL。
 *
 * - 默认是免认证的公开地址：只有 IsPublic + ACTIVE + 非高风险类型的资产能访问，
 *   其余一律 404。头像、封面这类要在任意页面展示的文件用它。
 * - `admin: true` 是后台地址：受 RBAC（`/api/v1/admin/assets/download/:objectKey:GET`）
 *   保护，可访问私有资产，仅供后台页面下载使用。
 */
export function assetUrl(objectKey: string, options?: { admin?: boolean }): string {
  const base = options?.admin ? '/api/v1/admin/assets/download' : '/api/v1/assets/download';
  return `${base}/${encodeURIComponent(objectKey)}`;
}
