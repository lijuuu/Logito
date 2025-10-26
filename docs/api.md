# Logito API Documentation

## Search Endpoint

### GET /search

Search logs with various filters and options.

#### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `message` | string | Search in log messages | `database connection` |
| `regex` | string | Regex pattern to match across all log fields | `.*error.*` |
| `level` | string | Filter by log level | `error`, `warn`, `info`, `debug` |
| `resourceId` | string | Filter by resource ID | `server-001` |
| `traceId` | string | Filter by trace ID | `trace-123` |
| `spanId` | string | Filter by span ID | `span-456` |
| `commit` | string | Filter by commit hash | `abc123` |
| `parentResourceId` | string | Filter by parent resource ID | `parent-789` |
| `startTime` | string | Start time (RFC3339) | `2023-01-01T00:00:00Z` |
| `endTime` | string | End time (RFC3339) | `2023-12-31T23:59:59Z` |
| `page` | integer | Page number (default: 1) | `1` |
| `limit` | integer | Results per page (1-1000, default: 10) | `50` |

#### Search Examples

**Basic message search:**
```
GET /search?message=database
```

**Phrase search:**
```
GET /search?message=database%20connection
```

**Regex search across all fields:**
```
GET /search?regex=.*error.*
```

**Semi-complex regex example - Find logs with specific patterns:**
```
GET /search?regex=^parent-.*$
```
This regex finds logs with parent resource IDs starting with "parent-".
```

**Combined filters:**
```
GET /search?level=error&message=timeout&limit=25
```

#### Response Format

```json
{
  "total": 150,
  "page": 1,
  "limit": 10,
  "totalPages": 15,
  "hasNext": true,
  "hasPrev": false,
  "took": 5,
  "results": [
    {
      "_id": "log_123",
      "_source": {
        "id": 123,
        "level": "error",
        "message": "Database connection failed",
        "resourceId": "server-001",
        "timestamp": "2023-01-01T12:00:00Z",
        "traceId": "trace-123",
        "spanId": "span-456",
        "commit": "abc123",
        "metadata": {
          "parentResourceId": "parent-789"
        }
      }
    }
  ]
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `total` | integer | Total number of matching logs |
| `page` | integer | Current page number |
| `limit` | integer | Results per page |
| `totalPages` | integer | Total number of pages |
| `hasNext` | boolean | Whether there's a next page |
| `hasPrev` | boolean | Whether there's a previous page |
| `took` | integer | Search time in milliseconds |
| `results` | array | Array of log entries |

#### Error Responses

**Invalid regex pattern:**
```json
{
  "error": "Invalid regex pattern: error parsing regexp: missing closing ]: `[invalid`"
}
```

**Invalid pagination:**
```json
{
  "error": "page must be a positive integer"
}
```

## Health Endpoint

### GET /health

Check service health status.

#### Response

```json
{
  "service": "query-interface",
  "status": "healthy",
  "timestamp": "2023-01-01T12:00:00Z"
}
```

## Log Ingestion Endpoint

### POST /logs

Ingest log entries.

#### Request Body

```json
[
  {
    "level": "info",
    "message": "Application started successfully",
    "resourceId": "server-001",
    "timestamp": "2023-01-01T12:00:00Z",
    "traceId": "trace-123",
    "spanId": "span-456",
    "commit": "abc123",
    "metadata": {
      "parentResourceId": "parent-789"
    }
  }
]
```

#### Response

```json
{
  "message": "logs processed",
  "total": 1,
  "valid": 1,
  "invalid": 0,
  "timestamp": "2023-01-01T12:00:00Z"
}
```
