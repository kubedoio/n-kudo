import { useEffect, useRef } from 'react';
import { useExecutionLogs, queryKeys } from '@/api/hooks';
import { LoadingSpinner, Badge, EmptyState } from '@/components/common';
import { useQueryClient } from '@tanstack/react-query';
import { Terminal } from 'lucide-react';


interface ExecutionLogViewerProps {
  executionId: string;
}

export function ExecutionLogViewer({ executionId }: ExecutionLogViewerProps) {
  const logsEndRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();
  
  // Use real API hook
  const { data: logs, isLoading, error } = useExecutionLogs(executionId, 100);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs?.length]);

  const getSeverityBadge = (severity: string) => {
    const variantMap: Record<string, 'default' | 'info' | 'warning' | 'error'> = {
      DEBUG: 'default',
      INFO: 'info',
      WARN: 'warning',
      WARNING: 'warning',
      ERROR: 'error',
    };

    return (
      <Badge variant={variantMap[severity] || 'default'} size="sm">
        {severity}
      </Badge>
    );
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <EmptyState
        icon="error"
        title="Failed to load logs"
        description={error.message}
        action={{
          label: 'Retry',
          onClick: () => {
            queryClient.invalidateQueries({
              queryKey: queryKeys.executionLogs(executionId),
            });
          },
        }}
      />
    );
  }

  // Show empty state if no logs
  if (!logs || logs.length === 0) {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3 border-b border-gray-200 pb-3">
          <Terminal className="w-5 h-5 text-gray-500" />
          <div>
            <h4 className="text-sm font-medium text-gray-900">Execution Logs</h4>
            <p className="text-xs text-gray-500">0 entries</p>
          </div>
        </div>
        <EmptyState
          icon="file"
          title="No logs yet"
          description="Logs will appear here once the execution starts"
          iconSize="md"
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-gray-200 pb-3">
        <div className="flex items-center gap-3">
          <Terminal className="w-5 h-5 text-gray-500" />
          <div>
            <h4 className="text-sm font-medium text-gray-900">Execution Logs</h4>
            <p className="text-xs text-gray-500">
              {logs.length} entries
            </p>
          </div>
        </div>
      </div>

      {/* Log Entries */}
      <div className="bg-gray-900 rounded-lg overflow-hidden">
        <div className="max-h-96 overflow-y-auto p-4 font-mono text-sm">
          {logs.map((log) => (
            <div
              key={log.id}
              className="flex gap-3 py-1 hover:bg-gray-800/50 transition-colors"
            >
              {/* Timestamp */}
              <span className="text-gray-500 flex-shrink-0 w-20">
                {formatTimestamp(log.emitted_at)}
              </span>

              {/* Severity */}
              <span className="flex-shrink-0 w-16">
                {getSeverityBadge(log.severity)}
              </span>

              {/* Message */}
              <span className="text-gray-300 break-all">
                {log.message}
              </span>
            </div>
          ))}
          <div ref={logsEndRef} />
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 text-xs text-gray-500 pt-2">
        <div className="flex items-center gap-1">
          <Badge variant="default" size="sm">DEBUG</Badge>
          <span>Debug</span>
        </div>
        <div className="flex items-center gap-1">
          <Badge variant="info" size="sm">INFO</Badge>
          <span>Info</span>
        </div>
        <div className="flex items-center gap-1">
          <Badge variant="warning" size="sm">WARN</Badge>
          <span>Warning</span>
        </div>
        <div className="flex items-center gap-1">
          <Badge variant="error" size="sm">ERROR</Badge>
          <span>Error</span>
        </div>
      </div>
    </div>
  );
}
