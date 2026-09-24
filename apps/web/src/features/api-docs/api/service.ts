import { apiDownload } from '@/lib/api-client';
import type { OpenApiDocument } from './types';

/** The generated OpenAPI document (served raw, not in the { code, data } envelope). */
export async function getOpenApiDocument(): Promise<OpenApiDocument> {
  const { blob } = await apiDownload('/v1/admin/openapi.json', 'openapi.json');
  return JSON.parse(await blob.text()) as OpenApiDocument;
}
