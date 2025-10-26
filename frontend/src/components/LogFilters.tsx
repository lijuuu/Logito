import React, { useState, useEffect } from 'react';
import { SearchFilters } from '../types';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { useMetadata } from '../hooks/useLogs';
import { Search, Filter, X, HelpCircle } from 'lucide-react';
import { convertLocalToUTC, convertUTCToLocal } from '../utils/dateUtils';
import { validateRegex } from '../utils';

interface LogFiltersProps {
  filters: SearchFilters;
  onFiltersChange: (filters: SearchFilters) => void;
  onSearch: () => void;
  loading?: boolean;
}

export const LogFilters: React.FC<LogFiltersProps> = ({
  filters,
  onFiltersChange,
  onSearch,
  loading = false,
}) => {
  const [localFilters, setLocalFilters] = useState<SearchFilters>(() => ({
    ...filters,
    startTime: filters.startTime ? convertUTCToLocal(filters.startTime) : '',
    endTime: filters.endTime ? convertUTCToLocal(filters.endTime) : '',
  }));
  const [regexError, setRegexError] = useState<string | null>(null);
  const [showRegexHelp, setShowRegexHelp] = useState(false);
  const { data: metadata, isLoading: metadataLoading } = useMetadata();

  useEffect(() => {
    setLocalFilters({
      ...filters,
      startTime: filters.startTime ? convertUTCToLocal(filters.startTime) : '',
      endTime: filters.endTime ? convertUTCToLocal(filters.endTime) : '',
    });
  }, [filters]);

  const handleFilterChange = (key: keyof SearchFilters, value: string | number | undefined) => {
    const newFilters = { ...localFilters, [key]: value };
    setLocalFilters(newFilters);

    // Validate regex when regex field changes
    if (key === 'regex' && typeof value === 'string') {
      const validation = validateRegex(value);
      setRegexError(validation.isValid ? null : validation.error || 'Invalid regex pattern');
    }
  };

  const handleSearch = () => {
    // Don't search if regex is invalid
    if (localFilters.regex && regexError) {
      return;
    }

    // Convert local dates to UTC before sending to backend
    const filtersWithUTC = {
      ...localFilters,
      startTime: localFilters.startTime ? convertLocalToUTC(localFilters.startTime) : undefined,
      endTime: localFilters.endTime ? convertLocalToUTC(localFilters.endTime) : undefined,
    };
    onFiltersChange(filtersWithUTC);
    onSearch();
  };

  const handleClear = () => {
    const clearedFilters: SearchFilters = {
      page: 1,
      limit: 10,
    };
    setLocalFilters(clearedFilters);
    setRegexError(null);
    onFiltersChange(clearedFilters);
    onSearch();
  };

  const hasActiveFilters = Object.entries(localFilters).some(
    ([key, value]) => key !== 'page' && key !== 'limit' && value !== undefined && value !== ''
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Filter className="h-5 w-5" />
          Search Filters
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {/* Message Search */}
          <div className="sm:col-span-2 lg:col-span-2 xl:col-span-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">Message Search</label>
              <Input
                placeholder="Search in log messages..."
                value={localFilters.message || ''}
                onChange={(e) => handleFilterChange('message', e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
              />
            </div>
          </div>

          {/* Regex Search */}
          <div className="sm:col-span-2 lg:col-span-2 xl:col-span-2">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium">Regex Search</label>
                <div className="relative">
                  <HelpCircle
                    className="h-4 w-4 text-gray-400 cursor-help"
                    onMouseEnter={() => setShowRegexHelp(true)}
                    onMouseLeave={() => setShowRegexHelp(false)}
                  />
                  {showRegexHelp && (
                    <div className="absolute z-10 w-72 sm:w-80 p-3 mt-2 bg-white border border-gray-200 rounded-lg shadow-lg text-xs left-0 sm:left-auto right-0">
                      <div className="space-y-2">
                        <div className="font-semibold text-gray-800">Regex Examples:</div>
                        <div className="space-y-1">
                          <div><code className="bg-gray-100 px-1 rounded">.*error.*</code> - Contains "error"</div>
                          <div><code className="bg-gray-100 px-1 rounded">^[0-9]&#123;3,4&#125;$</code> - 3-4 digit numbers</div>
                          <div><code className="bg-gray-100 px-1 rounded">parent-.*</code> - Starts with "parent-"</div>
                          <div><code className="bg-gray-100 px-1 rounded">^[A-Z][a-z]&#123;0,3&#125;$</code> - 1-4 chars, capital start</div>
                          <div><code className="bg-gray-100 px-1 rounded">DB_CONN.*</code> - Starts with "DB_CONN"</div>
                          <div><code className="bg-gray-100 px-1 rounded">1\.0\.0</code> - Exact version "1.0.0"</div>
                        </div>
                        <div className="text-gray-600 pt-1 border-t">
                          Searches across: message, level, resourceId, traceId, spanId, commit, and all metadata fields
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
              <Input
                placeholder="Enter regex pattern to search across all fields: .*error.*"
                value={localFilters.regex || ''}
                onChange={(e) => handleFilterChange('regex', e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                className={regexError ? 'border-red-500' : ''}
              />
              {regexError && (
                <p className="text-sm text-red-500">{regexError}</p>
              )}
            </div>
          </div>

          {/* Level Filter */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Log Level</label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              value={localFilters.level || ''}
              onChange={(e) => handleFilterChange('level', e.target.value)}
            >
              <option value="">All levels</option>
              <option value="ERROR">ERROR</option>
              <option value="WARN">WARN</option>
              <option value="INFO">INFO</option>
              <option value="DEBUG">DEBUG</option>
            </select>
          </div>

          {/* Resource ID */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Resource ID</label>
            <div className="flex gap-2">
              <Input
                placeholder="Filter by resource ID"
                value={localFilters.resourceId || ''}
                onChange={(e) => handleFilterChange('resourceId', e.target.value)}
                className="flex-1"
              />
              <select
                className="flex h-9 w-32 rounded-md border border-input bg-background px-2 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                value=""
                onChange={(e) => {
                  if (e.target.value) {
                    handleFilterChange('resourceId', e.target.value);
                    e.target.value = ''; // Reset dropdown after selection
                  }
                }}
              >
                <option value="">Choose...</option>
                {metadata?.metadata?.resourceIds?.slice(0, 20).map((item) => (
                  <option key={item.key} value={item.key}>
                    {item.key}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Trace ID */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Trace ID</label>
            <Input
              placeholder="Filter by trace ID"
              value={localFilters.traceId || ''}
              onChange={(e) => handleFilterChange('traceId', e.target.value)}
            />
          </div>

          {/* Span ID */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Span ID</label>
            <Input
              placeholder="Filter by span ID"
              value={localFilters.spanId || ''}
              onChange={(e) => handleFilterChange('spanId', e.target.value)}
            />
          </div>

          {/* Commit */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Commit</label>
            <Input
              placeholder="Filter by commit hash"
              value={localFilters.commit || ''}
              onChange={(e) => handleFilterChange('commit', e.target.value)}
            />
          </div>

          {/* Parent Resource ID */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Parent Resource ID</label>
            <Input
              placeholder="Filter by parent resource ID"
              value={localFilters.parentResourceId || ''}
              onChange={(e) => handleFilterChange('parentResourceId', e.target.value)}
            />
          </div>

          {/* Start Time */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Start Time (Local)</label>
            <Input
              type="datetime-local"
              value={localFilters.startTime || ''}
              onChange={(e) => handleFilterChange('startTime', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Enter local time, will be converted to UTC</p>
          </div>

          {/* End Time */}
          <div className="space-y-2">
            <label className="text-sm font-medium">End Time (Local)</label>
            <Input
              type="datetime-local"
              value={localFilters.endTime || ''}
              onChange={(e) => handleFilterChange('endTime', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Enter local time, will be converted to UTC</p>
          </div>

          {/* Limit */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Results per page</label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              value={localFilters.limit || 10}
              onChange={(e) => handleFilterChange('limit', parseInt(e.target.value))}
            >
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </select>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 mt-6">
          <Button onClick={handleSearch} disabled={loading} className="w-full sm:w-auto">
            <Search className="h-4 w-4 mr-2" />
            {loading ? 'Searching...' : 'Search'}
          </Button>

          {hasActiveFilters && (
            <Button variant="outline" onClick={handleClear} className="w-full sm:w-auto">
              <X className="h-4 w-4 mr-2" />
              Clear Filters
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
};