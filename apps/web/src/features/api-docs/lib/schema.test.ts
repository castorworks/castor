import { describe, expect, it } from 'vitest';
import type { OpenApiDocument } from '../api/types';
import {
  constraints,
  expandable,
  groupOperations,
  permissionOf,
  requiredAccess,
  resolve,
  successBody,
  typeLabel
} from './schema';

const doc: OpenApiDocument = {
  openapi: '3.1.0',
  info: { title: 'T', version: 'v1' },
  tags: [{ name: 'Users', description: 'User management' }, { name: 'Auth' }],
  components: {
    schemas: {
      Role: { type: 'object', properties: { code: { type: 'string' } } },
      UserResp: {
        type: 'object',
        properties: { roles: { type: 'array', items: { $ref: '#/components/schemas/Role' } } }
      }
    }
  },
  paths: {
    '/api/v1/auth/login': { post: { tags: ['Auth'], summary: 'Sign in', security: [] } },
    '/api/v1/admin/users/{id}': {
      delete: { tags: ['Users'], summary: 'Delete a user', security: [{ cookieAuth: [] }] },
      get: {
        tags: ['Users'],
        summary: 'Get a user',
        security: [{ cookieAuth: [] }],
        responses: {
          '200': {
            content: {
              'application/json': {
                schema: {
                  type: 'object',
                  properties: {
                    code: { type: 'integer' },
                    data: { $ref: '#/components/schemas/UserResp' }
                  }
                }
              }
            }
          }
        }
      }
    },
    '/api/v1/account/info': {
      get: { tags: ['Account'], summary: 'Me', security: [{ cookieAuth: [] }] }
    }
  }
};

describe('api docs schema helpers', () => {
  it('labels types compactly', () => {
    expect(typeLabel({ type: 'string', format: 'date-time' })).toBe('string (date-time)');
    expect(typeLabel({ type: 'array', items: { $ref: '#/components/schemas/Role' } })).toBe(
      'Role[]'
    );
    expect(typeLabel({ anyOf: [{ type: 'integer' }, { type: 'null' }] })).toBe('integer | null');
    expect(
      typeLabel({
        type: 'object',
        additionalProperties: { type: 'array', items: { $ref: '#/components/schemas/Role' } }
      })
    ).toBe('map<string, Role[]>');
    expect(typeLabel(undefined)).toBe('any');
  });

  it('summarizes constraints', () => {
    expect(constraints({ type: 'string', minLength: 3, maxLength: 100 })).toEqual(['3–100 chars']);
    expect(constraints({ type: 'array', maxItems: 5 })).toEqual(['≤ 5 items']);
    expect(constraints({ type: 'integer', minimum: 0 })).toEqual([]);
    expect(constraints({ type: 'string', enum: ['A', 'B'] })).toEqual(['A | B']);
  });

  it('resolves refs and finds expandable objects', () => {
    expect(resolve({ $ref: '#/components/schemas/Role' }, doc)?.properties?.code).toBeDefined();
    const roles = doc.components!.schemas!.UserResp.properties!.roles;
    expect(Object.keys(expandable(roles, doc)?.properties ?? {})).toEqual(['code']);
    expect(expandable({ type: 'string' }, doc)).toBeUndefined();
  });

  it('groups by document tag order and filters by text', () => {
    const groups = groupOperations(doc, '');
    expect(groups.map((g) => g.tag)).toEqual(['Users', 'Auth', 'Account']);
    expect(groups[0].entries.map((e) => e.method)).toEqual(['get', 'delete']);
    expect(groups[0].description).toBe('User management');
    expect(groupOperations(doc, 'sign').map((g) => g.tag)).toEqual(['Auth']);
  });

  it('unwraps the success envelope and derives access', () => {
    const [users] = groupOperations(doc, 'get a user');
    const entry = users.entries[0];
    expect(successBody(entry)?.schema?.$ref).toBe('#/components/schemas/UserResp');
    expect(requiredAccess(entry)).toBe('permission');
    expect(permissionOf(entry)).toBe('/api/v1/admin/users/:id:GET');
    const [auth] = groupOperations(doc, 'sign in');
    expect(requiredAccess(auth.entries[0])).toBe('public');
    const [account] = groupOperations(doc, '/account/');
    expect(requiredAccess(account.entries[0])).toBe('session');
  });
});
