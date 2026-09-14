import {
  createSearchParamsCache,
  createSerializer,
  parseAsInteger,
  parseAsString
} from 'nuqs/server';

export const searchParams = {
  page: parseAsInteger.withDefault(1),
  perPage: parseAsInteger.withDefault(10),
  search: parseAsString,
  asset: parseAsString,
  user: parseAsString,
  username: parseAsString,
  operator: parseAsString,
  logType: parseAsString,
  target: parseAsString,
  name: parseAsString, // alias for search (used by products)
  sort: parseAsString,
  gender: parseAsString,
  category: parseAsString,
  accountSource: parseAsString,
  role: parseAsString,
  module: parseAsString,
  requestMethod: parseAsString,
  status: parseAsString,
  title: parseAsString,
  type: parseAsString,
  level: parseAsString,
  action: parseAsString,
  loginMethod: parseAsString,
  success: parseAsString,
  result: parseAsString
  // advanced filter
  // filters: getFiltersStateParser().withDefault([]),
  // joinOperator: parseAsStringEnum(['and', 'or']).withDefault('and')
};

export const searchParamsCache = createSearchParamsCache(searchParams);
export const serialize = createSerializer(searchParams);
