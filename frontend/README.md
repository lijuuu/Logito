# Logito Query Interface Frontend

A modern React TypeScript application for querying and analyzing application logs.

## Features

- **Advanced Search**: Full-text search across log messages, levels, and resource IDs
- **Comprehensive Filtering**: Filter by level, resource ID, trace ID, span ID, commit, and date ranges
- **Real-time Stats**: Live statistics and counts for different log levels
- **Pagination**: Efficient pagination for large result sets
- **Detailed View**: Modal with complete log entry details and JSON export
- **Responsive Design**: Works on desktop and mobile devices
- **TanStack Query**: Efficient data fetching with caching and background updates

## Tech Stack

- React 18 with TypeScript
- TanStack Query for data fetching
- Tailwind CSS for styling
- Lucide React for icons
- Date-fns for date formatting

## Getting Started

1. Install dependencies:
```bash
npm install
```

2. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your API URL
```

3. Start the development server:
```bash
npm start
```

4. Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

## API Integration

The frontend integrates with the Logito Query Interface API endpoints:

- `GET /search` - Search logs with filters
- `GET /metadata` - Get available filter options
- `GET /counts` - Get log statistics
- `GET /logs/:id` - Get specific log entry
- `GET /health` - Health check

## Available Scripts

- `npm start` - Runs the app in development mode
- `npm build` - Builds the app for production
- `npm test` - Launches the test runner
- `npm eject` - Ejects from Create React App (one-way operation)

## Project Structure

```
src/
├── components/          # React components
│   ├── ui/             # Reusable UI components
│   ├── LogFilters.tsx  # Search and filter interface
│   ├── LogEntry.tsx    # Individual log entry display
│   ├── Pagination.tsx  # Pagination component
│   ├── StatsCards.tsx  # Statistics display
│   └── LogDetailsModal.tsx # Detailed log view
├── hooks/              # Custom React hooks
│   └── useLogs.ts      # TanStack Query hooks
├── services/           # API service layer
│   └── api.ts          # API client
├── types/              # TypeScript type definitions
│   └── index.ts        # API and component types
├── utils/              # Utility functions
│   └── index.ts        # Helper functions
├── App.tsx             # Main application component
└── index.tsx           # Application entry point
```
