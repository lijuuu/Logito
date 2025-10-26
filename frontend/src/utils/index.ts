import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { format, parseISO, isValid } from 'date-fns';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatTimestamp(timestamp: string): string {
  try {
    const date = parseISO(timestamp);
    if (!isValid(date)) {
      return timestamp;
    }
    return format(date, 'MMM dd, yyyy HH:mm:ss');
  } catch {
    return timestamp;
  }
}

export function validateRegex(pattern: string): { isValid: boolean; error?: string } {
  if (!pattern.trim()) {
    return { isValid: true };
  }

  try {
    new RegExp(pattern);
    return { isValid: true };
  } catch (error) {
    return {
      isValid: false,
      error: error instanceof Error ? error.message : 'Invalid regex pattern'
    };
  }
}

export function formatRelativeTime(timestamp: string): string {
  try {
    const date = parseISO(timestamp);
    if (!isValid(date)) {
      return timestamp;
    }

    const now = new Date();
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (diffInSeconds < 60) {
      return `${diffInSeconds}s ago`;
    } else if (diffInSeconds < 3600) {
      const minutes = Math.floor(diffInSeconds / 60);
      return `${minutes}m ago`;
    } else if (diffInSeconds < 86400) {
      const hours = Math.floor(diffInSeconds / 3600);
      return `${hours}h ago`;
    } else {
      const days = Math.floor(diffInSeconds / 86400);
      return `${days}d ago`;
    }
  } catch {
    return timestamp;
  }
}

export function getLogLevelColor(level: string): string {
  switch (level.toLowerCase()) {
    case 'error':
      return 'text-red-600 bg-red-50 border-red-200';
    case 'warn':
    case 'warning':
      return 'text-yellow-600 bg-yellow-50 border-yellow-200';
    case 'info':
      return 'text-blue-600 bg-blue-50 border-blue-200';
    case 'debug':
      return 'text-gray-600 bg-gray-50 border-gray-200';
    case 'trace':
      return 'text-purple-600 bg-purple-50 border-purple-200';
    default:
      return 'text-gray-600 bg-gray-50 border-gray-200';
  }
}

export function truncateText(text: string, maxLength: number = 100): string {
  if (text.length <= maxLength) {
    return text;
  }
  return text.substring(0, maxLength) + '...';
}

export function buildSearchUrl(filters: Record<string, any>): string {
  const params = new URLSearchParams();

  Object.entries(filters).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      params.append(key, String(value));
    }
  });

  return params.toString();
}

export function parseSearchUrl(searchParams: URLSearchParams): Record<string, string> {
  const filters: Record<string, string> = {};

  for (const [key, value] of searchParams.entries()) {
    filters[key] = value;
  }

  return filters;
}
