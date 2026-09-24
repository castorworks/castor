import type { UseMutationOptions } from '@tanstack/react-query';

async function runBoth<TArgs extends unknown[]>(
  first: ((...args: TArgs) => Promise<unknown> | unknown) | undefined,
  second: ((...args: TArgs) => Promise<unknown> | unknown) | undefined,
  ...args: TArgs
) {
  await first?.(...args);
  await second?.(...args);
}

export function mergeMutationOptions<
  TData = unknown,
  TError = Error,
  TVariables = void,
  TOnMutateResult = unknown
>(
  base: UseMutationOptions<TData, TError, TVariables, TOnMutateResult>,
  override: Omit<UseMutationOptions<TData, TError, TVariables, TOnMutateResult>, 'mutationFn'>
): UseMutationOptions<TData, TError, TVariables, TOnMutateResult> {
  return {
    ...base,
    ...override,
    mutationFn: base.mutationFn,
    mutationKey: override.mutationKey ?? base.mutationKey,
    meta: { ...base.meta, ...override.meta },
    onMutate: async (...args) => {
      const baseResult = await base.onMutate?.(...args);
      await override.onMutate?.(...args);
      return baseResult as TOnMutateResult;
    },
    onSuccess: async (...args) => {
      await runBoth(base.onSuccess, override.onSuccess, ...args);
    },
    onError: async (...args) => {
      await runBoth(base.onError, override.onError, ...args);
    },
    onSettled: async (...args) => {
      await runBoth(base.onSettled, override.onSettled, ...args);
    }
  };
}
