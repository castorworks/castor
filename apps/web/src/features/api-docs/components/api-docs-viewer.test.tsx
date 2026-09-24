import fs from 'node:fs';
import path from 'node:path';
import { renderToStaticMarkup } from 'react-dom/server';
import { NextIntlClientProvider } from 'next-intl';
import { describe, expect, it, vi } from 'vitest';
import messages from '../../../../messages/en.json';
import type { OpenApiDocument } from '../api/types';
import { groupOperations } from '../lib/schema';
import { ApiDocsViewer, OperationDetails } from './api-docs-viewer';

// The document the backend generates and commits (apps/api/openapi.json).
const doc = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '../../../../../api/openapi.json'), 'utf8')
) as OpenApiDocument;

vi.mock('@tanstack/react-query', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-query')>()),
  useQuery: () => ({ data: doc, isPending: false, error: null })
}));

function render(node: React.ReactNode) {
  return renderToStaticMarkup(
    <NextIntlClientProvider locale='en' messages={messages}>
      {node}
    </NextIntlClientProvider>
  );
}

function entry(method: string, apiPath: string) {
  const found = groupOperations(doc, '')
    .flatMap((g) => g.entries)
    .find((e) => e.method === method && e.path === apiPath);
  if (!found) throw new Error(`${method} ${apiPath} not in the document`);
  return found;
}

describe('ApiDocsViewer with the generated document', () => {
  it('lists every operation grouped by tag', () => {
    const html = render(<ApiDocsViewer />);
    const total = Object.values(doc.paths).reduce((n, item) => n + Object.keys(item).length, 0);
    expect(total).toBeGreaterThan(100);
    expect(html).toContain(`${total} endpoints`);
    for (const tag of doc.tags ?? []) {
      expect(html).toContain(`>${tag.name}</h3>`);
    }
    expect(html).toContain('/api/v1/admin/users/{id}');
    expect(html).toContain('Download OpenAPI JSON');
  });

  it('renders an admin operation with its permission, body and response', () => {
    const html = render(
      <OperationDetails entry={entry('put', '/api/v1/admin/users/{id}')} doc={doc} />
    );
    expect(html).toContain('/api/v1/admin/users/:id:PUT');
    expect(html).toContain('UserPutReq');
    expect(html).toContain('UserResp');
    expect(html).toContain('≤ 100 chars'); // name: binding max=100
    expect(html).toContain('application/json');
  });

  it('renders multipart uploads and file downloads', () => {
    const upload = render(
      <OperationDetails entry={entry('post', '/api/v1/admin/users/import')} doc={doc} />
    );
    expect(upload).toContain('multipart/form-data');
    expect(upload).toContain('password');
    const exported = render(
      <OperationDetails entry={entry('get', '/api/v1/admin/users/export')} doc={doc} />
    );
    expect(exported).toContain('A file download.');
    expect(exported).toContain('filters');
  });

  it('marks public endpoints', () => {
    const html = render(<OperationDetails entry={entry('post', '/api/v1/auth/login')} doc={doc} />);
    expect(html).toContain('No sign-in needed');
    expect(html).toContain('LoginRequest');
  });
});
