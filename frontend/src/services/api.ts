import {
  LogEntry,
  SearchResponse,
  MetadataResponse,
  CountsResponse,
  SearchFilters,
  LogEntryResponse,
  HealthResponse,
  SyncStatusResponse,
} from '../types';
import { logger } from '../utils/logger';

const API_BASE_URL = 'http://localhost:4000';

class ApiService {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
    logger.info('API Service initialized', { baseUrl });
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const startTime = Date.now();

    logger.logApiRequest(options?.method || 'GET', url, options?.body);

    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
        ...options,
      });

      const duration = Date.now() - startTime;
      logger.logApiResponse(options?.method || 'GET', url, response.status, duration);

      if (!response.ok) {
        const error = new Error(`API request failed: ${response.status} ${response.statusText}`);
        logger.logApiError(options?.method || 'GET', url, error);
        throw error;
      }

      return response.json();
    } catch (error) {
      const duration = Date.now() - startTime;
      logger.logApiError(options?.method || 'GET', url, error as Error);
      throw error;
    }
  }

  private buildQueryString(params: Record<string, any>): string {
    const searchParams = new URLSearchParams();

    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        searchParams.append(key, String(value));
      }
    });

    return searchParams.toString();
  }

  async searchLogs(filters: SearchFilters): Promise<SearchResponse> {
    logger.logUserAction('searchLogs', filters);
    const queryString = this.buildQueryString(filters);
    return this.request<SearchResponse>(`/search?${queryString}`);
  }

  async getMetadata(filters: Partial<SearchFilters> = {}): Promise<MetadataResponse> {
    logger.logUserAction('getMetadata', filters);
    const queryString = this.buildQueryString(filters);
    return this.request<MetadataResponse>(`/metadata?${queryString}`);
  }

  async getCounts(filters: Partial<SearchFilters> = {}): Promise<CountsResponse> {
    logger.logUserAction('getCounts', filters);
    const queryString = this.buildQueryString(filters);
    return this.request<CountsResponse>(`/counts?${queryString}`);
  }

  async getLogEntry(id: string): Promise<LogEntryResponse> {
    logger.logUserAction('getLogEntry', { id });
    return this.request<LogEntryResponse>(`/logs/${id}`);
  }

  async getHealth(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health');
  }

  async getSyncStatus(): Promise<SyncStatusResponse> {
    return this.request<SyncStatusResponse>('/sync-status');
  }

  async forceRefresh(): Promise<{ message: string; processedCount: number; timestamp: number }> {
    logger.logUserAction('forceRefresh');
    return this.request<{ message: string; processedCount: number; timestamp: number }>('/force-refresh', {
      method: 'POST',
    });
  }
}

export const apiService = new ApiService();
export default apiService;
