import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { useCounts } from '../hooks/useLogs';
import { SearchFilters } from '../types';
import { Activity, AlertTriangle, Info, Bug } from 'lucide-react';

interface StatsCardsProps {
  filters: Partial<SearchFilters>;
}

export const StatsCards: React.FC<StatsCardsProps> = ({ filters }) => {
  const { data: counts, isLoading } = useCounts(filters);

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        {[...Array(4)].map((_, i) => (
          <Card key={i} className="animate-pulse">
            <CardHeader className="pb-2">
              <div className="h-4 bg-gray-200 rounded w-3/4"></div>
            </CardHeader>
            <CardContent>
              <div className="h-8 bg-gray-200 rounded w-1/2"></div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  const totalCount = counts?.counts?.total || 0;
  const levelCounts = counts?.counts?.byLevel || [];

  const errorCount = levelCounts.find(l => l.key.toLowerCase() === 'error')?.docCount || 0;
  const warnCount = levelCounts.find(l => l.key.toLowerCase() === 'warn' || l.key.toLowerCase() === 'warning')?.docCount || 0;
  const infoCount = levelCounts.find(l => l.key.toLowerCase() === 'info')?.docCount || 0;
  const debugCount = levelCounts.find(l => l.key.toLowerCase() === 'debug')?.docCount || 0;

  const stats = [
    {
      title: 'Total Logs (Est.)',
      value: totalCount.toLocaleString(),
      icon: Activity,
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
    },
    {
      title: 'Errors',
      value: errorCount.toLocaleString(),
      icon: AlertTriangle,
      color: 'text-red-600',
      bgColor: 'bg-red-50',
    },
    {
      title: 'Warnings',
      value: warnCount.toLocaleString(),
      icon: Info,
      color: 'text-yellow-600',
      bgColor: 'bg-yellow-50',
    },
    {
      title: 'Debug',
      value: debugCount.toLocaleString(),
      icon: Bug,
      color: 'text-gray-600',
      bgColor: 'bg-gray-50',
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
      {stats.map((stat) => {
        const Icon = stat.icon;
        return (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium text-gray-600">
                {stat.title}
              </CardTitle>
              <div className={`p-2 rounded-full ${stat.bgColor}`}>
                <Icon className={`h-4 w-4 ${stat.color}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-gray-900">
                {stat.value}
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
};
