import { useEffect, useRef, useState } from 'react';

interface LogEntry {
  id?: number;
  level: string;
  message: string;
  resourceId: string;
  timestamp: string;
  traceId?: string;
  spanId?: string;
  commit?: string;
  metadata?: any;
}

interface BatchData {
  entries: LogEntry[];
  timestamp: number;
}

export const useWebSocketStream = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [batches, setBatches] = useState<BatchData[]>([]);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const connect = () => {
      const ws = new WebSocket('ws://localhost:3000/ws');

      ws.onopen = () => {
        setIsConnected(true);
      };

      ws.onmessage = (event) => {
        try {
          const batch = JSON.parse(event.data);
          setBatches(prev => [...prev.slice(-99), batch]);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        setTimeout(connect, 3000);
      };

      ws.onerror = () => {
        setIsConnected(false);
      };

      wsRef.current = ws;
    };

    connect();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  return { isConnected, batches };
};

