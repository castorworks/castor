import { describe, expect, it, vi } from 'vitest';
import { mergeMutationOptions } from './mutation-utils';

describe('mergeMutationOptions', () => {
  it('runs both base and override success callbacks in order', async () => {
    const calls: string[] = [];
    const base = {
      mutationFn: vi.fn(async (value: number) => value + 1),
      onSuccess: vi.fn(async () => {
        calls.push('base-success');
      })
    };
    const override = {
      onSuccess: vi.fn(async () => {
        calls.push('override-success');
      })
    };

    const merged = mergeMutationOptions(base, override);
    const data = await merged.mutationFn?.(1, {} as never);

    await merged.onSuccess?.(data!, 1, undefined, {} as never);

    expect(calls).toEqual(['base-success', 'override-success']);
    expect(base.onSuccess).toHaveBeenCalledOnce();
    expect(override.onSuccess).toHaveBeenCalledOnce();
  });

  it('runs both base and override error and settled callbacks', async () => {
    const calls: string[] = [];
    const error = new Error('boom');
    const base = {
      mutationFn: vi.fn(async () => {
        throw error;
      }),
      onError: vi.fn(async () => {
        calls.push('base-error');
      }),
      onSettled: vi.fn(async () => {
        calls.push('base-settled');
      })
    };
    const override = {
      onError: vi.fn(async () => {
        calls.push('override-error');
      }),
      onSettled: vi.fn(async () => {
        calls.push('override-settled');
      })
    };

    const merged = mergeMutationOptions(base, override);

    await merged.onError?.(error, undefined, undefined, {} as never);
    await merged.onSettled?.(undefined, error, undefined, undefined, {} as never);

    expect(calls).toEqual(['base-error', 'override-error', 'base-settled', 'override-settled']);
  });
});
