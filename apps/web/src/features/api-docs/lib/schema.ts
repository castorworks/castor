import type { OpenApiDocument, OperationEntry, Schema } from '../api/types';

const REF_PREFIX = '#/components/schemas/';

/** Component name of a `$ref`, e.g. `UserResp`. */
export function refName(ref: string): string {
  return ref.startsWith(REF_PREFIX) ? ref.slice(REF_PREFIX.length) : ref;
}

/** Follow `$ref`s (bounded, so a broken document cannot loop forever). */
export function resolve(schema: Schema | undefined, doc: OpenApiDocument): Schema | undefined {
  let current = schema;
  for (let i = 0; current?.$ref && i < 16; i++) {
    current = doc.components?.schemas?.[refName(current.$ref)];
  }
  return current;
}

/** `T | null` is written as `anyOf: [T, {type: null}]`; split it back. */
export function unwrapNullable(schema: Schema): { schema: Schema; nullable: boolean } {
  if (schema.anyOf?.length === 2) {
    const other = schema.anyOf.find((s) => s.type !== 'null');
    if (other && schema.anyOf.some((s) => s.type === 'null')) {
      return { schema: other, nullable: true };
    }
  }
  return { schema, nullable: false };
}

/** A compact type label: `string`, `UserResp`, `Role[]`, `map<string, DictItemResp[]>`. */
export function typeLabel(schema: Schema | undefined): string {
  if (!schema) return 'any';
  const { schema: inner, nullable } = unwrapNullable(schema);
  let label: string;
  if (inner.$ref) {
    label = refName(inner.$ref);
  } else if (inner.type === 'array') {
    label = `${typeLabel(inner.items)}[]`;
  } else if (
    inner.type === 'object' &&
    inner.additionalProperties &&
    typeof inner.additionalProperties === 'object'
  ) {
    label = `map<string, ${typeLabel(inner.additionalProperties)}>`;
  } else if (inner.type === 'string' && inner.format) {
    label = `string (${inner.format})`;
  } else if (inner.type) {
    label = inner.type;
  } else {
    label = 'any';
  }
  return nullable ? `${label} | null` : label;
}

/** Validation hints shown next to a field: `3–100 chars`, `≤ 5 items`, `A | B`. */
export function constraints(schema: Schema | undefined): string[] {
  if (!schema) return [];
  const s = unwrapNullable(schema).schema;
  const out: string[] = [];
  const range = (min?: number, max?: number, unit = '') => {
    if (min !== undefined && max !== undefined) out.push(`${min}–${max}${unit}`);
    else if (min !== undefined) out.push(`≥ ${min}${unit}`);
    else if (max !== undefined) out.push(`≤ ${max}${unit}`);
  };
  range(s.minLength, s.maxLength, ' chars');
  range(s.minItems, s.maxItems, ' items');
  if (s.minimum !== 0 || s.maximum !== undefined) range(s.minimum, s.maximum);
  if (s.enum?.length) out.push(s.enum.map(String).join(' | '));
  if (s.const !== undefined) out.push(`= ${String(s.const)}`);
  return out;
}

/** The object schema to expand under a field, if any (objects, arrays of objects, maps of objects). */
export function expandable(schema: Schema | undefined, doc: OpenApiDocument): Schema | undefined {
  if (!schema) return undefined;
  const inner = unwrapNullable(schema).schema;
  const target =
    inner.type === 'array'
      ? inner.items
      : inner.type === 'object' && typeof inner.additionalProperties === 'object'
        ? inner.additionalProperties
        : inner;
  const resolved = target ? resolve(target, doc) : undefined;
  return resolved?.properties && Object.keys(resolved.properties).length > 0 ? resolved : undefined;
}

const METHOD_ORDER = ['get', 'post', 'put', 'patch', 'delete'];

/** Operations grouped by tag, in document tag order, filtered by a free-text query. */
export function groupOperations(
  doc: OpenApiDocument,
  search: string
): { tag: string; description?: string; entries: OperationEntry[] }[] {
  const needle = search.trim().toLowerCase();
  const byTag = new Map<string, OperationEntry[]>();
  for (const [path, item] of Object.entries(doc.paths)) {
    for (const method of METHOD_ORDER) {
      const operation = item[method];
      if (!operation) continue;
      const haystack = `${method} ${path} ${operation.summary ?? ''}`.toLowerCase();
      if (needle && !haystack.includes(needle)) continue;
      const tag = operation.tags?.[0] ?? 'Other';
      byTag.set(tag, [...(byTag.get(tag) ?? []), { method, path, operation }]);
    }
  }
  const order = (doc.tags ?? []).map((t) => t.name);
  return [...byTag.entries()]
    .toSorted(([a], [b]) => (order.indexOf(a) + 1 || 999) - (order.indexOf(b) + 1 || 999))
    .map(([tag, entries]) => ({
      tag,
      description: doc.tags?.find((t) => t.name === tag)?.description,
      entries: entries.toSorted(
        (a, b) =>
          a.path.localeCompare(b.path) ||
          METHOD_ORDER.indexOf(a.method) - METHOD_ORDER.indexOf(b.method)
      )
    }));
}

/** The `data` schema inside the `{ code, data, message }` success envelope, or the raw body. */
export function successBody(
  entry: OperationEntry
): { mediaType: string; schema?: Schema } | undefined {
  const ok = entry.operation.responses?.['200'];
  const [mediaType, media] = Object.entries(ok?.content ?? {})[0] ?? [];
  if (!mediaType) return undefined;
  const envelope = media?.schema;
  const data = envelope?.properties?.data;
  if (mediaType === 'application/json' && data && envelope?.properties?.code) {
    return { mediaType, schema: data };
  }
  return { mediaType, schema: envelope };
}

/** Required permission of an admin operation (`{path}:{METHOD}`), derived like the backend does. */
export function requiredAccess(entry: OperationEntry): 'public' | 'session' | 'permission' {
  if (!entry.operation.security?.length) return 'public';
  return entry.path.startsWith('/api/v1/admin/') ? 'permission' : 'session';
}

/** OpenAPI `{id}` back to the Gin route template `:id`, which is how permissions are named. */
export function permissionOf(entry: OperationEntry): string {
  return `${entry.path.replace(/\{([^}]+)\}/g, ':$1')}:${entry.method.toUpperCase()}`;
}
