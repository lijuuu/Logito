import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiService } from '../services/api';
import { SyncStatusResponse } from '../types';
import { logger } from '../utils/logger';

export const useSyncStatus = () => {
  return useQuery({
    queryKey: ['syncStatus'],
    queryFn: () => apiService.getSyncStatus(),
    staleTime: 5000, // Data is considered stale after 1 second
    refetchInterval: 5000, // Refetch every 1 second
    refetchIntervalInBackground: true, // Continue refetching even when tab is not active
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
};

export const useForceRefresh = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => apiService.forceRefresh(),
    onSuccess: (data) => {
      logger.logUserAction('forceRefreshCompleted', data);
      // Invalidate and refetch sync status after successful force refresh
      queryClient.invalidateQueries({ queryKey: ['syncStatus'] });
    },
    onError: (error) => {
      logger.logUserAction('forceRefreshError', { error: error.message });
    },
  });
};
