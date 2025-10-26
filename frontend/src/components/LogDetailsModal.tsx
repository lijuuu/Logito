import React from 'react';
import { useLogEntry } from '../hooks/useLogs';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { formatTimestamp, getLogLevelColor } from '../utils';
import { X, Copy, Download } from 'lucide-react';

interface LogDetailsModalProps {
  logId: string | null;
  onClose: () => void;
}

export const LogDetailsModal: React.FC<LogDetailsModalProps> = ({ logId, onClose }) => {
  const { data: logData, isLoading, error } = useLogEntry(logId || '');

  const handleCopy = async () => {
    if (logData?.logEntry) {
      try {
        await navigator.clipboard.writeText(JSON.stringify(logData.logEntry, null, 2));
      } catch (err) {
        console.error('Failed to copy:', err);
      }
    }
  };

  const handleDownload = () => {
    if (logData?.logEntry) {
      const dataStr = JSON.stringify(logData.logEntry, null, 2);
      const dataBlob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(dataBlob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `log-entry-${logData.logEntry.id}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    }
  };

  if (!logId) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-2 sm:p-4 z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[95vh] sm:max-h-[90vh] overflow-hidden">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between p-4 sm:p-6 border-b border-gray-200 gap-3">
          <h2 className="text-lg sm:text-xl font-semibold text-gray-900">Log Entry Details</h2>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleCopy} className="flex-1 sm:flex-none">
              <Copy className="h-4 w-4 mr-2" />
              <span className="hidden sm:inline">Copy</span>
            </Button>
            <Button variant="outline" size="sm" onClick={handleDownload} className="flex-1 sm:flex-none">
              <Download className="h-4 w-4 mr-2" />
              <span className="hidden sm:inline">Download</span>
            </Button>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="p-4 sm:p-6 overflow-y-auto max-h-[calc(95vh-120px)] sm:max-h-[calc(90vh-120px)]">
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          )}

          {error && (
            <div className="text-center py-8">
              <p className="text-red-600">Failed to load log entry details</p>
            </div>
          )}

          {logData?.logEntry && (
            <div className="space-y-6">
              {/* Header Info */}
              <Card>
                <CardHeader>
                  <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3">
                    <div className="flex items-center gap-2 sm:gap-3">
                      <span
                        className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium border ${getLogLevelColor(
                          logData.logEntry.level
                        )}`}
                      >
                        {logData.logEntry.level.toUpperCase()}
                      </span>
                      <span className="text-sm text-gray-500 font-mono">
                        ID: {logData.logEntry.id}
                      </span>
                    </div>
                    <span className="text-sm text-gray-500">
                      {formatTimestamp(logData.logEntry.timestamp)}
                    </span>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-gray-900 leading-relaxed">
                    {logData.logEntry.message}
                  </p>
                </CardContent>
              </Card>

              {/* Basic Information */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Basic Information</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <label className="text-sm font-medium text-gray-500">Resource ID</label>
                      <p className="mt-1 font-mono text-sm text-gray-900 break-all">
                        {logData.logEntry.resourceId}
                      </p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-500">Timestamp</label>
                      <p className="mt-1 text-sm text-gray-900 break-all">
                        {formatTimestamp(logData.logEntry.timestamp)}
                      </p>
                    </div>
                    {logData.logEntry.traceId && (
                      <div>
                        <label className="text-sm font-medium text-gray-500">Trace ID</label>
                        <p className="mt-1 font-mono text-sm text-gray-900 break-all">
                          {logData.logEntry.traceId}
                        </p>
                      </div>
                    )}
                    {logData.logEntry.spanId && (
                      <div>
                        <label className="text-sm font-medium text-gray-500">Span ID</label>
                        <p className="mt-1 font-mono text-sm text-gray-900 break-all">
                          {logData.logEntry.spanId}
                        </p>
                      </div>
                    )}
                    {logData.logEntry.commit && (
                      <div>
                        <label className="text-sm font-medium text-gray-500">Commit</label>
                        <p className="mt-1 font-mono text-sm text-gray-900 break-all">
                          {logData.logEntry.commit}
                        </p>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* Metadata */}
              {logData.logEntry.metadata && Object.keys(logData.logEntry.metadata).length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Metadata</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <pre className="bg-gray-50 p-4 rounded-md text-sm text-gray-700 overflow-x-auto">
                      {JSON.stringify(logData.logEntry.metadata, null, 2)}
                    </pre>
                  </CardContent>
                </Card>
              )}

              {/* Raw JSON */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Raw JSON</CardTitle>
                </CardHeader>
                <CardContent>
                  <pre className="bg-gray-50 p-4 rounded-md text-sm text-gray-700 overflow-x-auto">
                    {JSON.stringify(logData.logEntry, null, 2)}
                  </pre>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
