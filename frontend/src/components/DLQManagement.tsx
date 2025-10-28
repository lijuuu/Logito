import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { Button } from './ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { apiService } from '../services/api';

interface DLQMessage {
  id: string;
  payload: string;
  reason: string;
  created_at: string;
}

interface DLQStats {
  count: number;
}

export const DLQManagement: React.FC = () => {
  const [dlqCount, setDlqCount] = useState<number>(0);
  const [messages, setMessages] = useState<DLQMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const { token, user } = useAuth();

  const refreshDLQ = async () => {
    if (!token) return;

    setIsLoading(true);
    try {
      // Fetch both count and messages
      const [countData, messagesData] = await Promise.all([
        apiService.getDLQCount(),
        apiService.getDLQMessages(10)
      ]);

      setDlqCount(countData.count);
      setMessages(messagesData.messages || []);
    } catch (err) {
      setError('Failed to refresh DLQ data');
    } finally {
      setIsLoading(false);
    }
  };


  const forceAddAllMessages = async () => {
    if (!token || user?.role !== 'admin') return;

    if (!confirm('Are you sure you want to force add all DLQ messages to PostgreSQL? This will bypass validation.')) {
      return;
    }

    try {
      await apiService.forceAddAllDLQMessages();
      // Refresh the data
      refreshDLQ();
    } catch (err) {
      setError('Failed to force add all messages');
    }
  };

  const clearDLQ = async () => {
    if (!token || user?.role !== 'admin') return;

    if (!confirm('Are you sure you want to clear all DLQ messages? This action cannot be undone.')) {
      return;
    }

    try {
      await apiService.clearDLQ();
      // Refresh the data
      refreshDLQ();
    } catch (err) {
      setError('Failed to clear DLQ');
    }
  };

  useEffect(() => {
    refreshDLQ();
  }, [token]);

  if (!token) {
    return <div>Please log in to access DLQ management.</div>;
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Dead Letter Queue Management</CardTitle>
          <CardDescription>
            Monitor and manage failed log entries
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center space-x-4">
              <Badge variant={dlqCount > 0 ? "destructive" : "secondary"}>
                {dlqCount} Failed Messages
              </Badge>
              <Button onClick={refreshDLQ} variant="outline" size="sm" disabled={isLoading}>
                {isLoading ? 'Refreshing...' : 'Refresh'}
              </Button>
            </div>
            {dlqCount > 0 && user?.role === 'admin' && (
              <div className="flex space-x-2">
                <Button onClick={forceAddAllMessages} variant="outline" size="sm">
                  Force Add All
                </Button>
                <Button onClick={clearDLQ} variant="destructive" size="sm">
                  Clear All
                </Button>
              </div>
            )}
          </div>

          {error && (
            <div className="text-red-600 text-sm mb-4">{error}</div>
          )}

          <div className="space-y-4">

            {messages.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-lg font-semibold">Recent Failed Messages</h3>
                {messages.map((message) => (
                  <Card key={message.id} className="p-4">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="text-sm text-gray-600 mb-2">
                          <strong>ID:</strong> {message.id}
                        </div>
                        <div className="text-sm text-gray-600 mb-2">
                          <strong>Reason:</strong> {message.reason}
                        </div>
                        <div className="text-sm text-gray-600 mb-2">
                          <strong>Created:</strong> {new Date(message.created_at).toLocaleString()}
                        </div>
                        <div className="text-sm text-gray-600">
                          <strong>Payload:</strong>
                          <pre className="mt-1 text-xs bg-gray-100 p-2 rounded overflow-x-auto">
                            {message.payload}
                          </pre>
                        </div>
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            )}

            {messages.length === 0 && !isLoading && (
              <div className="text-center text-gray-500 py-8">
                No failed messages found
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
