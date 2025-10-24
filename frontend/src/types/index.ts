export interface LogEntry {
  id: number;
  level: string;
  message: string;
  resourceId: string;
  timestamp: string;
  traceId?: string;
  spanId?: string;
  commit?: string;
  metadata?: Record<string, any>;
}

export interface SearchResponse {
  total: number;
  page: number;
  limit: number;
  totalPages: number;
  hasNext: boolean;
  hasPrev: boolean;
  results: Array<{
    _index: string;
    _id: string;
    _score: number;
    _source: LogEntry;
  }>;
  took: number;
}

export interface MetadataResponse {
  metadata: {
    levels: Array<{ key: string; docCount: number }>;
    resourceIds: Array<{ key: string; docCount: number }>;
    traceIds: Array<{ key: string; docCount: number }>;
    spanIds: Array<{ key: string; docCount: number }>;
    commits: Array<{ key: string; docCount: number }>;
    parentResourceIds: Array<{ key: string; docCount: number }>;
    timestampRange: {
      min: string;
      max: string;
      count: number;
    };
  };
  took: number;
}

export interface CountsResponse {
  counts: {
    total: number;
    byLevel: Array<{ key: string; docCount: number }>;
    byResource: Array<{ key: string; docCount: number }>;
    hourly: Array<{ keyAsString: string; key: number; docCount: number }>;
    daily: Array<{ keyAsString: string; key: number; docCount: number }>;
  };
  took: number;
}

export interface SearchFilters {
  message?: string;
  level?: string;
  resourceId?: string;
  traceId?: string;
  spanId?: string;
  commit?: string;
  parentResourceId?: string;
  startTime?: string;
  endTime?: string;
  page?: number;
  limit?: number;
}

export interface LogEntryResponse {
  logEntry: LogEntry;
  took: number;
}

export interface HealthResponse {
  status: string;
  timestamp: string;
  service: string;
}

export interface SyncStatusResponse {
  status: string;
  totalInPostgres: number;
  indexedInPostgres: number;
  unindexedInPostgres: number;
  totalInES: number;
  remainingToIndex: number;
  workerHealthy: boolean;
  timestamp: string;
}
