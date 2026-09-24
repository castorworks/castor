'use client';

import { Fragment, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { saveFile } from '@/lib/download';
import { tagColorBg } from '@/lib/tag-color';
import { cn } from '@/lib/utils';
import { openApiDocumentQueryOptions } from '../api/queries';
import type { OpenApiDocument, OperationEntry, Parameter, Schema } from '../api/types';
import {
  constraints,
  expandable,
  groupOperations,
  permissionOf,
  requiredAccess,
  successBody,
  typeLabel
} from '../lib/schema';

const METHOD_COLORS: Record<string, string> = {
  get: 'blue',
  post: 'green',
  put: 'orange',
  patch: 'purple',
  delete: 'red'
};

/** Nested object fields render this deep before stopping (recursive types). */
const MAX_DEPTH = 5;

/** Render `code spans` in the backend's English descriptions. */
function RichText({ text }: { text: string }) {
  return (
    <>
      {text.split(/(`[^`]+`)/g).map((part, i) =>
        part.startsWith('`') && part.endsWith('`') ? (
          <code key={i} className='bg-muted rounded px-1 font-mono text-xs'>
            {part.slice(1, -1)}
          </code>
        ) : (
          <Fragment key={i}>{part}</Fragment>
        )
      )}
    </>
  );
}

function MethodBadge({ method }: { method: string }) {
  return (
    <Badge variant='outline' className='w-16 justify-start font-mono uppercase'>
      <span
        aria-hidden
        className={cn('size-2 shrink-0 rounded-full', tagColorBg(METHOD_COLORS[method]))}
      />
      {method}
    </Badge>
  );
}

/** Fields of an object schema, with nested objects expandable. */
function SchemaFields({
  schema,
  doc,
  depth,
  seen
}: {
  schema: Schema;
  doc: OpenApiDocument;
  depth: number;
  seen: Set<Schema>;
}) {
  const t = useTranslations('apiDocs');
  const required = new Set(schema.required ?? []);
  const entries = Object.entries(schema.properties ?? {}).toSorted(([a], [b]) =>
    a.localeCompare(b)
  );
  if (entries.length === 0) {
    return <p className='text-muted-foreground text-xs'>{t('emptyObject')}</p>;
  }
  return (
    <ul className='divide-y rounded-md border'>
      {entries.map(([name, field]) => {
        const nested = depth < MAX_DEPTH ? expandable(field, doc) : undefined;
        const recursive = nested ? seen.has(nested) : false;
        const hints = constraints(field);
        const row = (
          <div className='flex flex-wrap items-baseline gap-x-3 gap-y-1 px-3 py-2 text-sm'>
            <span className='font-mono font-medium'>
              {name}
              {required.has(name) && (
                <span className='text-destructive' title={t('required')}>
                  *
                </span>
              )}
            </span>
            <span className='text-muted-foreground font-mono text-xs'>{typeLabel(field)}</span>
            {hints.map((hint) => (
              <span key={hint} className='text-muted-foreground text-xs'>
                {hint}
              </span>
            ))}
            {field.description && (
              <span className='text-muted-foreground basis-full text-xs'>
                <RichText text={field.description} />
              </span>
            )}
          </div>
        );
        if (!nested || recursive) {
          return <li key={name}>{row}</li>;
        }
        return (
          <li key={name}>
            <Collapsible>
              <CollapsibleTrigger className='hover:bg-muted/50 group flex w-full items-center text-left'>
                <Icons.chevronRight className='text-muted-foreground ml-2 size-4 shrink-0 transition-transform group-data-[state=open]:rotate-90' />
                <div className='flex-1'>{row}</div>
              </CollapsibleTrigger>
              <CollapsibleContent className='px-3 pb-3 pl-8'>
                <SchemaFields
                  schema={nested}
                  doc={doc}
                  depth={depth + 1}
                  seen={new Set([...seen, nested])}
                />
              </CollapsibleContent>
            </Collapsible>
          </li>
        );
      })}
    </ul>
  );
}

/** A request or response body: its type, then its fields when it is an object. */
function SchemaBlock({ schema, doc }: { schema?: Schema; doc: OpenApiDocument }) {
  const t = useTranslations('apiDocs');
  if (!schema || schema.type === 'null') {
    return <p className='text-muted-foreground text-sm'>{t('dataNull')}</p>;
  }
  const fields = expandable(schema, doc);
  return (
    <div className='space-y-2'>
      <p className='font-mono text-xs'>{typeLabel(schema)}</p>
      {fields && <SchemaFields schema={fields} doc={doc} depth={1} seen={new Set([fields])} />}
    </div>
  );
}

function ParametersTable({ parameters }: { parameters: Parameter[] }) {
  const t = useTranslations('apiDocs');
  return (
    <div className='overflow-x-auto rounded-md border'>
      <table className='w-full text-sm'>
        <thead className='bg-muted/50'>
          <tr className='text-left'>
            <th className='px-3 py-2 font-medium'>{t('name')}</th>
            <th className='px-3 py-2 font-medium'>{t('in')}</th>
            <th className='px-3 py-2 font-medium'>{t('type')}</th>
            <th className='px-3 py-2 font-medium'>{t('description')}</th>
          </tr>
        </thead>
        <tbody>
          {parameters.map((p) => (
            <tr key={`${p.in}-${p.name}`} className='border-t align-top'>
              <td className='px-3 py-2 font-mono'>
                {p.name}
                {p.required && <span className='text-destructive'>*</span>}
              </td>
              <td className='text-muted-foreground px-3 py-2'>{p.in}</td>
              <td className='text-muted-foreground px-3 py-2 font-mono text-xs'>
                {typeLabel(p.schema)}
                {constraints(p.schema).map((hint) => (
                  <div key={hint}>{hint}</div>
                ))}
              </td>
              <td className='text-muted-foreground px-3 py-2 text-xs'>
                {p.description && <RichText text={p.description} />}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function OperationDetails({ entry, doc }: { entry: OperationEntry; doc: OpenApiDocument }) {
  const t = useTranslations('apiDocs');
  const { operation } = entry;
  const access = requiredAccess(entry);
  const body = operation.requestBody?.content ?? {};
  const [bodyType, bodyMedia] = Object.entries(body)[0] ?? [];
  const success = successBody(entry);
  const parameters = operation.parameters ?? [];
  // The permission line is already shown as its own field.
  const description = operation.description?.replace(/\n*Requires the permission `[^`]+`\.$/, '');

  return (
    <div className='space-y-5 px-4 pt-2 pb-5'>
      {description && (
        <p className='text-muted-foreground text-sm whitespace-pre-line'>
          <RichText text={description} />
        </p>
      )}
      <dl className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm'>
        <dt className='text-muted-foreground'>{t('access.label')}</dt>
        <dd>
          {access === 'permission' ? (
            <>
              {t('access.permission')}{' '}
              <code className='bg-muted rounded px-1 font-mono text-xs'>{permissionOf(entry)}</code>
            </>
          ) : (
            t(`access.${access}`)
          )}
        </dd>
        {operation.operationId && (
          <>
            <dt className='text-muted-foreground'>{t('operationId')}</dt>
            <dd className='font-mono text-xs'>{operation.operationId}</dd>
          </>
        )}
      </dl>

      <section className='space-y-2'>
        <h4 className='text-sm font-semibold'>{t('parameters')}</h4>
        {parameters.length > 0 ? (
          <ParametersTable parameters={parameters} />
        ) : (
          <p className='text-muted-foreground text-sm'>{t('noParameters')}</p>
        )}
      </section>

      {bodyType && (
        <section className='space-y-2'>
          <h4 className='text-sm font-semibold'>
            {t('requestBody')}{' '}
            <span className='text-muted-foreground font-mono text-xs'>{bodyType}</span>
          </h4>
          <SchemaBlock schema={bodyMedia?.schema} doc={doc} />
        </section>
      )}

      <section className='space-y-2'>
        <h4 className='text-sm font-semibold'>
          {t('response')}{' '}
          {success && (
            <span className='text-muted-foreground font-mono text-xs'>{success.mediaType}</span>
          )}
        </h4>
        {success?.mediaType === 'application/json' ? (
          <SchemaBlock schema={success.schema} doc={doc} />
        ) : (
          <p className='text-muted-foreground text-sm'>{t('fileResponse')}</p>
        )}
      </section>
    </div>
  );
}

/** Read-only browser for the OpenAPI document the backend generates from its route catalog. */
export function ApiDocsViewer() {
  const t = useTranslations('apiDocs');
  const query = useQuery(openApiDocumentQueryOptions());
  const [search, setSearch] = useState('');
  const doc = query.data;
  const groups = useMemo(() => (doc ? groupOperations(doc, search) : []), [doc, search]);

  if (query.isPending) {
    return <div className='bg-muted h-96 animate-pulse rounded-lg' />;
  }
  if (!doc) {
    return (
      <p role='alert' className='text-destructive text-sm'>
        {query.error?.message || t('loadFailed')}
      </p>
    );
  }

  const total = Object.values(doc.paths).reduce((n, item) => n + Object.keys(item).length, 0);
  const download = () => {
    try {
      saveFile({
        blob: new Blob([JSON.stringify(doc, null, 2)], { type: 'application/json' }),
        filename: 'castor-openapi.json'
      });
    } catch {
      toast.error(t('downloadFailed'));
    }
  };

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-center gap-3'>
        <div className='relative min-w-0 flex-1 sm:max-w-sm'>
          <Icons.search className='text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t('search')}
            aria-label={t('search')}
            className='pl-8'
          />
        </div>
        <span className='text-muted-foreground text-sm'>
          {doc.info.title} {doc.info.version} · {t('endpoints', { count: total })}
        </span>
        <Button variant='outline' size='sm' className='ml-auto' onClick={download}>
          <Icons.download /> {t('download')}
        </Button>
      </div>

      {doc.info.description && (
        <p className='text-muted-foreground max-w-4xl text-sm'>
          <RichText text={doc.info.description} />
        </p>
      )}

      {groups.length === 0 && <p className='text-muted-foreground text-sm'>{t('noResults')}</p>}

      {groups.map((group) => (
        <section key={group.tag} className='space-y-2'>
          <div>
            <h3 className='text-base font-semibold'>{group.tag}</h3>
            {group.description && (
              <p className='text-muted-foreground text-sm'>
                <RichText text={group.description} />
              </p>
            )}
          </div>
          <div className='divide-y rounded-lg border'>
            {group.entries.map((entry) => (
              <Collapsible key={`${entry.method} ${entry.path}`}>
                <CollapsibleTrigger className='hover:bg-muted/50 group flex w-full flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2.5 text-left'>
                  <MethodBadge method={entry.method} />
                  <span className='min-w-0 font-mono text-sm break-all'>{entry.path}</span>
                  <span className='text-muted-foreground text-sm'>{entry.operation.summary}</span>
                  <Icons.chevronDown className='text-muted-foreground ml-auto size-4 shrink-0 transition-transform group-data-[state=open]:rotate-180' />
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <OperationDetails entry={entry} doc={doc} />
                </CollapsibleContent>
              </Collapsible>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
