import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

import { apiService } from '../services/api';
import { logger } from '../utils/logger';

export const useSyncStatus = () => {
  return useQuery({
    queryKey: ['syncStatus'],
    queryFn: () => apiService.getSyncStatus(),
    refetchInterval: 3000, 
    refetchIntervalInBackground: true, 
    retry: 3,
  });
};

export const useForceRefresh = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => apiService.forceRefresh(),
    onSuccess: (data) => {
      logger.logUserAction('forceRefreshCompleted', data);
      
      queryClient.invalidateQueries({ queryKey: ['syncStatus'] });
    },
    onError: (error) => {
      logger.logUserAction('forceRefreshError', { error: error.message });
    },
  });
};

export const useResetIndexing = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => apiService.resetIndexingStatus(),
    onSuccess: (data) => {
      logger.logUserAction('resetIndexingCompleted', data);
      
      queryClient.invalidateQueries({ queryKey: ['syncStatus'] });
    },
    onError: (error) => {
      logger.logUserAction('resetIndexingError', { error: error.message });
    },
  });
};
