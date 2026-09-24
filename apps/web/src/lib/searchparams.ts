import { createSearchParamsCache, parseAsInteger, parseAsString } from 'nuqs/server';

/**
 * URL 查询参数的中央登记表。
 *
 * nuqs 的 cache 需要一个扁平对象，但扁平对象会让所有模块的参数挤在一起：
 * `status`、`type`、`category` 这类通用名一旦跨模块复用，谁在用、能不能改、
 * 删掉会不会影响别人就没人说得清了。所以这里按模块分组声明，再合并成扁平表：
 * 分组只是可读性与归属，真正的约束由 searchparams.test.ts 负责——
 * 它会拒绝跨组重复定义，也会揪出没有任何使用方的死参数。
 *
 * 新增模块的筛选参数时：加到自己的分组里；确实要与别的模块共用同一个 URL 参数，
 * 就放进 sharedParams 并在注释里写清共用方。
 */

/** 所有分页表格共用的分页、搜索与排序状态。 */
const sharedParams = {
  page: parseAsInteger.withDefault(1),
  perPage: parseAsInteger.withDefault(10),
  sort: parseAsString,
  // search：resources、users 的关键词框
  search: parseAsString,
  // status：resources、users、assets 的启用/禁用筛选
  status: parseAsString,
  // category：resources（资源分类）、assets（资产分类）
  category: parseAsString,
  // user：login-histories、sessions 的用户名筛选
  user: parseAsString
};

const auditLogParams = {
  operator: parseAsString,
  logType: parseAsString,
  target: parseAsString,
  success: parseAsString
};

const loginHistoryParams = {
  loginMethod: parseAsString,
  result: parseAsString
};

const sessionParams = {
  ip: parseAsString
};

const userParams = {
  accountSource: parseAsString,
  department: parseAsString
};

const resourceParams = {
  module: parseAsString
};

const assetParams = {
  asset: parseAsString,
  // scope：缺省只列资产库（LIBRARY），业务附件需显式筛选
  scope: parseAsString
};

const jobParams = {
  // job / trigger / runStatus：执行记录表的任务、触发方式、状态筛选
  job: parseAsString,
  trigger: parseAsString,
  runStatus: parseAsString
};

const notificationParams = {
  title: parseAsString,
  type: parseAsString,
  level: parseAsString
};

/** 分组登记表，测试按这个结构检查重复与失效。 */
export const searchParamGroups = {
  shared: sharedParams,
  auditLogs: auditLogParams,
  loginHistories: loginHistoryParams,
  sessions: sessionParams,
  users: userParams,
  resources: resourceParams,
  assets: assetParams,
  notifications: notificationParams,
  jobs: jobParams
} as const;

export const searchParams = {
  ...sharedParams,
  ...auditLogParams,
  ...loginHistoryParams,
  ...sessionParams,
  ...userParams,
  ...resourceParams,
  ...assetParams,
  ...notificationParams,
  ...jobParams
};

export const searchParamsCache = createSearchParamsCache(searchParams);
