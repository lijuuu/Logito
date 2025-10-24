import { useState, useEffect } from 'react';
import { LogFilters } from './components/LogFilters';
import { LogEntry } from './components/LogEntry';
import { Pagination } from './components/Pagination';
import { StatsCards } from './components/StatsCards';
import { LogDetailsModal } from './components/LogDetailsModal';
import { IndexingStatus } from './components/IndexingStatus';
import { useLogs, useInvalidateLogs } from './hooks/useLogs';
import { SearchFilters } from './types';
import { Search, RefreshCw, AlertCircle, Database, Activity } from 'lucide-react';
import { logger } from './utils/logger';
import { Button } from './components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './components/ui/card';
import { Badge } from './components/ui/badge';

function App() {
  const [filters, setFilters] = useState<SearchFilters>({
    page: 1,
    limit: 10,
  });
  const [selectedLogId, setSelectedLogId] = useState<string | null>(null);
  const [searchTriggered, setSearchTriggered] = useState(false);

  const { data: logsData, isLoading, error, refetch } = useLogs(filters);
  const { invalidateAll } = useInvalidateLogs();

  // Load initial data
  useEffect(() => {
    logger.info('App component mounted');
    if (!searchTriggered) {
      setSearchTriggered(true);
      refetch();
    }
  }, [searchTriggered, refetch]);

  const handleFiltersChange = (newFilters: SearchFilters) => {
    logger.logUserAction('filtersChanged', newFilters);
    setFilters(newFilters);
  };

  const handleSearch = () => {
    logger.logUserAction('searchTriggered', filters);
    setSearchTriggered(true);
    refetch();
  };

  const handlePageChange = (page: number) => {
    logger.logUserAction('pageChanged', { page });
    setFilters(prev => ({ ...prev, page }));
  };

  const handleRefresh = () => {
    logger.logUserAction('refreshTriggered');
    invalidateAll();
    refetch();
  };

  const handleViewLogDetails = (id: string) => {
    logger.logUserAction('viewLogDetails', { id });
    setSelectedLogId(id);
  };

  const handleCloseModal = () => {
    logger.logUserAction('closeModal');
    setSelectedLogId(null);
  };

  const totalPages = logsData?.totalPages || 0;

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold tracking-tight">Logito</h1>
              <p className="mt-2 text-muted-foreground">
                Search and analyze your application logs with powerful filtering and real-time insights.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <Button variant="outline" onClick={handleRefresh}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Refresh
              </Button>
            </div>
          </div>
        </div>

        {/* Stats Cards */}
        <StatsCards filters={filters} />

        {/* Indexing Status */}
        <IndexingStatus />

        {/* Filters */}
        <LogFilters
          filters={filters}
          onFiltersChange={handleFiltersChange}
          onSearch={handleSearch}
          loading={isLoading}
        />

        {/* Results */}
        <div className="space-y-4">
          {/* Results Header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Search className="h-5 w-5 text-muted-foreground" />
              <h2 className="text-lg font-semibold">Search Results</h2>
              {logsData && (
                <Badge variant="secondary">
                  {logsData.total.toLocaleString()} total
                </Badge>
              )}
            </div>
            {logsData && (
              <div className="text-sm text-muted-foreground">
                Query took {logsData.took}ms
              </div>
            )}
          </div>

          {/* Loading State */}
          {isLoading && (
            <Card>
              <CardContent className="flex items-center justify-center py-12">
                <div className="flex items-center gap-3">
                  <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
                  <span className="text-muted-foreground">Searching logs...</span>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Error State */}
          {error && (
            <Card>
              <CardContent className="flex items-center justify-center py-12">
                <div className="text-center">
                  <AlertCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
                  <h3 className="text-lg font-medium mb-2">Search Failed</h3>
                  <p className="text-muted-foreground mb-4">
                    {error instanceof Error ? error.message : 'An unexpected error occurred'}
                  </p>
                  <Button onClick={handleRefresh}>
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Try Again
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Results */}
          {logsData && !isLoading && !error && (
            <>
              {logsData.results.length === 0 ? (
                <Card>
                  <CardContent className="text-center py-12">
                    <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-lg font-medium mb-2">No logs found</h3>
                    <p className="text-muted-foreground">
                      Try adjusting your search filters or search terms.
                    </p>
                  </CardContent>
                </Card>
              ) : (
                <>
                  <div className="space-y-4">
                    {logsData.results.map((result) => (
                      <LogEntry
                        key={result._id}
                        logEntry={result._source}
                        onViewDetails={handleViewLogDetails}
                      />
                    ))}
                  </div>

                  {/* Pagination */}
                  <Pagination
                    currentPage={filters.page || 1}
                    totalPages={totalPages}
                    onPageChange={handlePageChange}
                    totalItems={logsData.total}
                    itemsPerPage={filters.limit || 10}
                  />
                </>
              )}
            </>
          )}
        </div>
      </div>

      {/* Log Details Modal */}
      <LogDetailsModal
        logId={selectedLogId}
        onClose={handleCloseModal}
      />
    </div>
  );
}

export default App;
