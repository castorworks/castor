type SortingStateItem = {
  id: string;
  desc?: boolean;
};

function parseSort(sort: string): SortingStateItem | undefined {
  try {
    const parsed = JSON.parse(sort) as SortingStateItem[];
    return parsed[0];
  } catch {
    return undefined;
  }
}

export function buildCastorOrder(
  sort: string | undefined,
  orderFields: Record<string, string>
): string | undefined {
  if (!sort) return undefined;

  const sortItem = parseSort(sort);
  const field = sortItem ? orderFields[sortItem.id] : undefined;
  if (!field) return undefined;

  return `${field} ${sortItem?.desc ? 'desc' : 'asc'}`;
}
