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
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-4 sm:gap-6 mb-6">
        {[...Array(6)].map((_, i) => (
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

  // Debug logging
  console.log('Counts data:', counts);
  console.log('Level counts:', levelCounts);

  const fatalCount = levelCounts.find(l => l.key === 'FATAL')?.doc_count || 0;
  const errorCount = levelCounts.find(l => l.key === 'ERROR')?.doc_count || 0;
  const warnCount = levelCounts.find(l => l.key === 'WARN')?.doc_count || 0;
  const infoCount = levelCounts.find(l => l.key === 'INFO')?.doc_count || 0;
  const debugCount = levelCounts.find(l => l.key === 'DEBUG')?.doc_count || 0;

  // Debug logging for individual counts
  console.log('Fatal count:', fatalCount);
  console.log('Error count:', errorCount);
  console.log('Warn count:', warnCount);
  console.log('Info count:', infoCount);
  console.log('Debug count:', debugCount);

  const stats = [
    {
      title: 'Total Logs (Est.)',
      value: totalCount.toLocaleString(),
      icon: Activity,
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
    },
    {
      title: 'Fatal',
      value: fatalCount.toLocaleString(),
      icon: AlertTriangle,
      color: 'text-red-800',
      bgColor: 'bg-red-100',
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
      title: 'Info',
      value: infoCount.toLocaleString(),
      icon: Info,
      color: 'text-green-600',
      bgColor: 'bg-green-50',
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
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-4 sm:gap-6 mb-6">
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
