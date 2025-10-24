import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiService } from '../services/api';
import { SearchFilters, LogEntry } from '../types';

export const useLogs = (filters: SearchFilters) => {
  return useQuery({
    queryKey: ['logs', filters],
    queryFn: () => apiService.searchLogs(filters),
    staleTime: 30000, // 30 seconds
    refetchOnWindowFocus: false,
  });
};

export const useLogEntry = (id: string) => {
  return useQuery({
    queryKey: ['logEntry', id],
    queryFn: () => apiService.getLogEntry(id),
    enabled: !!id,
    staleTime: 60000, // 1 minute
  });
};

export const useMetadata = (filters: Partial<SearchFilters> = {}) => {
  return useQuery({
    queryKey: ['metadata', filters],
    queryFn: () => apiService.getMetadata(filters),
    staleTime: 60000, // 1 minute
    refetchOnWindowFocus: false,
  });
};

export const useCounts = (filters: Partial<SearchFilters> = {}) => {
  return useQuery({
    queryKey: ['counts', filters],
    queryFn: () => apiService.getCounts(filters),
    staleTime: 30000, // 30 seconds
    refetchOnWindowFocus: false,
  });
};

export const useHealth = () => {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => apiService.getHealth(),
    refetchInterval: 30000, // 30 seconds
    staleTime: 10000, // 10 seconds
  });
};

export const useInvalidateLogs = () => {
  const queryClient = useQueryClient();

  return {
    invalidateAll: () => {
      queryClient.invalidateQueries({ queryKey: ['logs'] });
      queryClient.invalidateQueries({ queryKey: ['metadata'] });
      queryClient.invalidateQueries({ queryKey: ['counts'] });
    },
    invalidateLogs: () => {
      queryClient.invalidateQueries({ queryKey: ['logs'] });
    },
    invalidateMetadata: () => {
      queryClient.invalidateQueries({ queryKey: ['metadata'] });
    },
    invalidateCounts: () => {
      queryClient.invalidateQueries({ queryKey: ['counts'] });
    },
  };
};
