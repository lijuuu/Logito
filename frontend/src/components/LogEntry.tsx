import React, { useState } from 'react';
import { LogEntry as LogEntryType } from '../types';
import { Card, CardContent, CardHeader } from './ui/card';
import { Button } from './ui/button';
import { formatTimestamp, formatRelativeTime, getLogLevelColor, truncateText } from '../utils';
import { ChevronDown, ChevronRight, Copy, ExternalLink } from 'lucide-react';

interface LogEntryProps {
  logEntry: LogEntryType;
  onViewDetails?: (id: string) => void;
}

export const LogEntry: React.FC<LogEntryProps> = ({ logEntry, onViewDetails }) => {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(logEntry, null, 2));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const handleViewDetails = () => {
    if (onViewDetails) {
      onViewDetails(logEntry.id.toString());
    }
  };

  return (
    <Card className="mb-4 hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <button
              onClick={() => setExpanded(!expanded)}
              className="flex-shrink-0 p-1 hover:bg-gray-100 rounded"
            >
              {expanded ? (
                <ChevronDown className="h-4 w-4 text-gray-500" />
              ) : (
                <ChevronRight className="h-4 w-4 text-gray-500" />
              )}
            </button>

            <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2 min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${getLogLevelColor(
                    logEntry.level
                  )}`}
                >
                  {logEntry.level.toUpperCase()}
                </span>

                <span className="text-sm text-gray-500 font-mono">
                  #{logEntry.id}
                </span>
              </div>

              <span className="text-sm text-gray-500">
                {formatRelativeTime(logEntry.timestamp)}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-2 flex-shrink-0">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCopy}
              className="h-8 w-8 p-0"
            >
              <Copy className="h-4 w-4" />
            </Button>

            {onViewDetails && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleViewDetails}
                className="h-8 w-8 p-0"
              >
                <ExternalLink className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <div className="space-y-3">
          {/* Message */}
          <div>
            <p className="text-sm text-gray-900 leading-relaxed">
              {expanded ? logEntry.message : truncateText(logEntry.message, 200)}
            </p>
          </div>

          {/* Basic Info */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
            <div className="break-words">
              <span className="text-gray-500">Resource ID:</span>
              <span className="ml-2 font-mono text-gray-900 break-all">{logEntry.resourceId}</span>
            </div>
            <div className="break-words">
              <span className="text-gray-500">Timestamp:</span>
              <span className="ml-2 text-gray-900 break-all">{formatTimestamp(logEntry.timestamp)}</span>
            </div>
          </div>

          {/* Optional Fields */}
          {(logEntry.traceId || logEntry.spanId || logEntry.commit) && (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 text-sm">
              {logEntry.traceId && (
                <div className="break-words">
                  <span className="text-gray-500">Trace ID:</span>
                  <span className="ml-2 font-mono text-gray-900 break-all">{logEntry.traceId}</span>
                </div>
              )}
              {logEntry.spanId && (
                <div className="break-words">
                  <span className="text-gray-500">Span ID:</span>
                  <span className="ml-2 font-mono text-gray-900 break-all">{logEntry.spanId}</span>
                </div>
              )}
              {logEntry.commit && (
                <div className="break-words">
                  <span className="text-gray-500">Commit:</span>
                  <span className="ml-2 font-mono text-gray-900 break-all">{logEntry.commit}</span>
                </div>
              )}
            </div>
          )}

          {/* Metadata */}
          {logEntry.metadata && Object.keys(logEntry.metadata).length > 0 && (
            <div>
              <span className="text-sm text-gray-500">Metadata:</span>
              <div className="mt-1 p-3 bg-gray-50 rounded-md overflow-x-auto">
                <pre className="text-xs text-gray-700 whitespace-pre-wrap break-words">
                  {JSON.stringify(logEntry.metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>

        {copied && (
          <div className="mt-3 text-sm text-green-600">
            Copied to clipboard!
          </div>
        )}
      </CardContent>
    </Card>
  );
};
