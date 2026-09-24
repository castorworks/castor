import type { ResourceFilters } from './types';

/** 资源列表的 URL 状态，服务端与客户端各自读取后交给同一个构造函数。 */
export interface ResourceSearchParams {
  page: number;
  perPage: number;
  search?: string | null;
  category?: string | null;
  module?: string | null;
  status?: string | null;
  sort?: string | null;
}

/**
 * 把 URL 状态转换为请求筛选条件。
 *
 * 服务端预取与客户端查询必须落在同一个 queryKey 上，否则 HydrationBoundary
 * 交付的数据会因为 key 不一致被直接丢弃，页面在有筛选时退化成客户端二次请求。
 * 因此两侧共用这一个构造函数，而不是各写一份。
 */
export function buildResourceFilters(params: ResourceSearchParams): ResourceFilters {
  return {
    page: params.page,
    pageSize: params.perPage,
    ...(params.search ? { search: params.search } : {}),
    ...(params.category ? { category: params.category } : {}),
    ...(params.module ? { module: params.module } : {}),
    ...(params.sort ? { sort: params.sort } : {}),
    ...(params.status === 'enabled' ? { isEnabled: true } : {}),
    ...(params.status === 'disabled' ? { isEnabled: false } : {})
  };
}
