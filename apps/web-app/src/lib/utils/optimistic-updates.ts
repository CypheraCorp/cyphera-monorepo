import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { logger } from '@/lib/core/logger/logger-utils';

interface OptimisticUpdateOptions<TData, TVariables> {
  // Query key to update
  queryKey: unknown[];
  // Function to update the cache optimistically
  updateFn: (oldData: TData | undefined, variables: TVariables) => TData;
  // Optional: function to revert on error
  revertFn?: (oldData: TData | undefined, error: Error) => TData;
  // Optional: function to reconcile server response with optimistic update
  reconcileFn?: (optimisticData: TData, serverData: TData) => TData;
}

export function useOptimisticUpdate<TData = unknown, TVariables = unknown>() {
  const queryClient = useQueryClient();

  const update = useCallback(
    async (
      options: OptimisticUpdateOptions<TData, TVariables>,
      variables: TVariables,
      mutationFn: () => Promise<TData>
    ) => {
      const { queryKey, updateFn, revertFn, reconcileFn } = options;

      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey });

      // Snapshot the previous value
      const previousData = queryClient.getQueryData<TData>(queryKey);

      // Optimistically update to the new value
      queryClient.setQueryData<TData>(queryKey, (old) => updateFn(old, variables));

      try {
        // Perform the mutation
        const serverData = await mutationFn();

        // Reconcile if function provided, otherwise use server data
        if (reconcileFn && previousData) {
          const optimisticData = queryClient.getQueryData<TData>(queryKey);
          if (optimisticData) {
            queryClient.setQueryData<TData>(queryKey, reconcileFn(optimisticData, serverData));
          }
        } else {
          queryClient.setQueryData<TData>(queryKey, serverData);
        }

        return serverData;
      } catch (error) {
        logger.error('Optimistic update failed:', { error });

        // Revert the optimistic update on error
        if (revertFn && error instanceof Error) {
          queryClient.setQueryData<TData>(queryKey, (old) => revertFn(old, error));
        } else {
          queryClient.setQueryData<TData>(queryKey, previousData);
        }

        throw error;
      }
    },
    [queryClient]
  );

  return { update };
}

// Example usage utilities for common patterns

// Optimistic delete from list
export function optimisticDeleteFromList<T extends { id: string }>(
  list: T[] | undefined,
  itemId: string
): T[] {
  return list?.filter((item) => item.id !== itemId) || [];
}

// Optimistic add to list
export function optimisticAddToList<T>(
  list: T[] | undefined,
  newItem: T,
  position: 'start' | 'end' = 'start'
): T[] {
  const currentList = list || [];
  return position === 'start' ? [newItem, ...currentList] : [...currentList, newItem];
}

// Optimistic update in list
export function optimisticUpdateInList<T extends { id: string }>(
  list: T[] | undefined,
  itemId: string,
  updates: Partial<T>
): T[] {
  return list?.map((item) => (item.id === itemId ? { ...item, ...updates } : item)) || [];
}

// Optimistic toggle boolean in list
export function optimisticToggleInList<T extends { id: string; [key: string]: unknown }>(
  list: T[] | undefined,
  itemId: string,
  field: keyof T
): T[] {
  return (
    list?.map((item) => (item.id === itemId ? { ...item, [field]: !item[field] } : item)) || []
  );
}
