import React, { useEffect, useRef, useState } from 'react';
import { useWebSocketStream } from '../hooks/useWebSocketStream';

interface LogStreamCanvasProps {
  width?: number;
  height?: number;
}

export const LogStreamCanvas: React.FC<LogStreamCanvasProps> = ({
  width = 800,
  height = 400
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const { isConnected, batches } = useWebSocketStream();
  const [logs, setLogs] = useState<any[]>([]);

  useEffect(() => {
    const allLogs = batches.flatMap(batch =>
      batch.entries.map(entry => ({
        ...entry,
        batchTime: batch.timestamp,
        id: Math.random().toString(36).substr(2, 9)
      }))
    );
    setLogs(allLogs.slice(-1000));
  }, [batches]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    canvas.width = width;
    canvas.height = height;

    const draw = () => {
      ctx.fillStyle = '#0a0a0a';
      ctx.fillRect(0, 0, width, height);

      const logHeight = 20;
      const maxLogs = Math.floor(height / logHeight);
      const visibleLogs = logs.slice(-maxLogs);

      visibleLogs.forEach((log, index) => {
        const y = height - (index + 1) * logHeight;

        const levelColors: { [key: string]: string } = {
          'error': '#ef4444',
          'warn': '#f59e0b',
          'info': '#3b82f6',
          'debug': '#10b981',
          'trace': '#8b5cf6'
        };

        const color = levelColors[log.level.toLowerCase()] || '#6b7280';

        ctx.fillStyle = color;
        ctx.fillRect(0, y, 4, logHeight);

        ctx.fillStyle = '#ffffff';
        ctx.font = '12px monospace';

        const timestamp = new Date(log.timestamp).toLocaleTimeString();
        const text = `[${timestamp}] ${log.level.toUpperCase()} ${log.message}`;

        ctx.fillText(text, 8, y + 14);
      });

      ctx.fillStyle = isConnected ? '#10b981' : '#ef4444';
      ctx.fillRect(width - 20, 10, 10, 10);

      ctx.fillStyle = '#ffffff';
      ctx.font = '10px monospace';
      ctx.fillText(`${logs.length} logs`, width - 100, 20);
    };

    draw();
  }, [logs, isConnected, width, height]);

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <div className="bg-gray-800 px-4 py-2 flex items-center justify-between">
        <h3 className="text-white font-mono text-sm">Live Log Stream</h3>
        <div className="flex items-center space-x-2">
          <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
          <span className="text-gray-300 text-xs">
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
      </div>
      <canvas
        ref={canvasRef}
        className="block"
        style={{ width: `${width}px`, height: `${height}px` }}
      />
    </div>
  );
};
