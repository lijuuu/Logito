import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Database, Activity, AlertTriangle, CheckCircle, RefreshCw, Loader2 } from 'lucide-react';
import { useSyncStatus, useForceRefresh } from '../hooks/useSyncStatus';

export const IndexingStatus: React.FC = () => {
  const { data: syncStatus, isLoading, error, refetch } = useSyncStatus();
  const { mutate: forceRefresh, isPending: isRefreshing } = useForceRefresh();

  const handleForceRefresh = () => {
    forceRefresh();
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'syncing':
        return <Activity className="h-5 w-5 text-blue-500 animate-pulse" />;
      case 'degraded':
        return <AlertTriangle className="h-5 w-5 text-yellow-500" />;
      case 'unhealthy':
        return <AlertTriangle className="h-5 w-5 text-red-500" />;
      default:
        return <Database className="h-5 w-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'bg-green-100 text-green-800 border-green-200';
      case 'syncing':
        return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'degraded':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'unhealthy':
        return 'bg-red-100 text-red-800 border-red-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Indexing Status
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-4">
            <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-2" />
            <p className="text-sm text-muted-foreground mb-4">
              {error instanceof Error ? error.message : 'Failed to fetch sync status'}
            </p>
            <Button variant="outline" onClick={() => refetch()} disabled={isLoading}>
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Indexing Status
          </div>
          {/* <Button
            variant="outline"
            size="sm"
            onClick={handleForceRefresh}
            disabled={isRefreshing}
          >
            {isRefreshing ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4 mr-2" />
            )}
            Force Refresh
          </Button> */}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {syncStatus ? (
          <div className="space-y-4">
            {/* Status Overview */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {getStatusIcon(syncStatus.status)}
                <span className="font-medium">Status:</span>
                <Badge className={getStatusColor(syncStatus.status)}>
                  {syncStatus.status.toUpperCase()}
                </Badge>
              </div>
              <div className="text-xs text-muted-foreground">
                Updated: {new Date(syncStatus.timestamp).toLocaleTimeString()}
              </div>
            </div>

            {/* Statistics Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-600">
                  {(syncStatus.indexedInPostgres || 0).toLocaleString()}
                </div>
                <div className="text-xs text-muted-foreground">Total Indexed</div>
              </div>

              <div className="text-center">
                <div className="text-2xl font-bold text-orange-600">
                  {(syncStatus.remainingToIndex || 0).toLocaleString()}
                </div>
                <div className="text-xs text-muted-foreground">Remaining</div>
              </div>

              <div className="text-center">
                <div className="text-2xl font-bold text-green-600">
                  {(syncStatus.totalInPostgres || 0).toLocaleString()}
                </div>
                <div className="text-xs text-muted-foreground">Total in DB</div>
              </div>

              <div className="text-center">
                <div className="flex items-center justify-center">
                  {syncStatus.workerHealthy ? (
                    <CheckCircle className="h-6 w-6 text-green-500" />
                  ) : (
                    <AlertTriangle className="h-6 w-6 text-red-500" />
                  )}
                </div>
                <div className="text-xs text-muted-foreground">Worker</div>
              </div>
            </div>

            {/* Progress Bar */}
            {(syncStatus.indexedInPostgres || 0) > 0 && (
              <div className="space-y-2">
                <div className="flex justify-between text-xs text-muted-foreground">
                  <span>Indexing Progress</span>
                  <span>
                    {Math.round(((syncStatus.indexedInPostgres || 0) / (syncStatus.totalInPostgres || 1)) * 100)}%
                  </span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                    style={{
                      width: `${((syncStatus.indexedInPostgres || 0) / (syncStatus.totalInPostgres || 1)) * 100}%`
                    }}
                  />
                </div>
              </div>
            )}

            {/* Status Messages */}
            {syncStatus.status === 'syncing' && (syncStatus.remainingToIndex || 0) > 0 && (
              <div className="text-sm text-blue-600 bg-blue-50 p-3 rounded-md">
                <Activity className="h-4 w-4 inline mr-2" />
                Currently indexing {(syncStatus.remainingToIndex || 0).toLocaleString()} logs...
              </div>
            )}

            {syncStatus.status === 'unhealthy' && (
              <div className="text-sm text-red-600 bg-red-50 p-3 rounded-md">
                <AlertTriangle className="h-4 w-4 inline mr-2" />
                Index worker is unhealthy. Check system status.
              </div>
            )}

            {syncStatus.status === 'healthy' && (syncStatus.remainingToIndex || 0) === 0 && (
              <div className="text-sm text-green-600 bg-green-50 p-3 rounded-md">
                <CheckCircle className="h-4 w-4 inline mr-2" />
                All logs are indexed and up to date.
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin mr-2" />
            <span className="text-muted-foreground">Loading indexing status...</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
