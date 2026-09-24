import { queryOptions } from '@tanstack/react-query';
import { getOpenApiDocument } from './service';

export const apiDocsKeys = {
  document: ['api-docs', 'document'] as const
};

export const openApiDocumentQueryOptions = () =>
  queryOptions({
    queryKey: apiDocsKeys.document,
    queryFn: getOpenApiDocument,
    // The document only changes when the API is redeployed.
    staleTime: Infinity
  });
