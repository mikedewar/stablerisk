# StableRisk API Documentation

This directory contains the OpenAPI 3.0 specification for the StableRisk API.

## Viewing the Documentation

### Swagger UI (Recommended)

Run Swagger UI locally:

```bash
docker run -p 8081:8080 \
  -e SWAGGER_JSON=/docs/openapi.yaml \
  -v $(pwd)/openapi.yaml:/docs/openapi.yaml \
  swaggerapi/swagger-ui
```

Then open http://localhost:8081 in your browser.

### Redoc

For a different documentation style:

```bash
docker run -p 8081:80 \
  -e SPEC_URL=/openapi.yaml \
  -v $(pwd)/openapi.yaml:/usr/share/nginx/html/openapi.yaml \
  redocly/redoc
```

### VS Code

Install the "OpenAPI (Swagger) Editor" extension and open `openapi.yaml`.

## API Overview

### Base URLs

- **Production**: `https://stablerisk.yourdomain.com/api/v1`
- **Development**: `http://localhost:8080/api/v1`

### Authentication

All endpoints (except health checks) require JWT authentication:

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-password"}'

# Use token in subsequent requests
curl -H "Authorization: Bearer <access-token>" \
  http://localhost:8080/api/v1/outliers
```

### WebSocket Connection

Real-time outlier notifications:

```javascript
const ws = new WebSocket('wss://stablerisk.yourdomain.com/api/v1/ws?token=<access-token>');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'outlier') {
    console.log('New outlier detected:', message.data);
  }
};

// Subscribe to specific outliers
ws.send(JSON.stringify({
  type: 'subscribe',
  filters: {
    severities: ['high', 'critical']
  }
}));
```

## Quick Start Examples

### List Outliers

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/outliers?severity=high&page=1&per_page=20"
```

### Get Outlier Details

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/outliers/<outlier-id>"
```

### Acknowledge Outlier

```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"notes": "Investigated - legitimate transaction"}' \
  "http://localhost:8080/api/v1/outliers/<outlier-id>/acknowledge"
```

### Trigger Manual Detection

```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"window_hours": 24}' \
  "http://localhost:8080/api/v1/detection/trigger"
```

### Get Statistics

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/statistics/transactions?window=24h"
```

## Response Codes

- `200 OK` - Success
- `201 Created` - Resource created
- `202 Accepted` - Request accepted for processing
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

## Rate Limits

- General API: 100 requests/second per IP
- Login endpoint: 5 requests/minute per IP
- WebSocket connections: 50 concurrent per IP

Exceeding rate limits returns `429 Too Many Requests`.

## Pagination

List endpoints support pagination:

```json
{
  "outliers": [...],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total_pages": 10,
    "total_items": 487
  }
}
```

## Error Responses

All errors follow a consistent format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "additional context"
  }
}
```

## Role-Based Access Control

### Roles

- **admin**: Full access to all endpoints
- **analyst**: Read/write access to outliers, can trigger detection
- **viewer**: Read-only access

### Permission Matrix

| Endpoint | Viewer | Analyst | Admin |
|----------|--------|---------|-------|
| GET /outliers | ✓ | ✓ | ✓ |
| POST /outliers/:id/acknowledge | ✗ | ✓ | ✓ |
| POST /detection/trigger | ✗ | ✓ | ✓ |
| GET /statistics/* | ✓ | ✓ | ✓ |
| POST /users | ✗ | ✗ | ✓ |

## Generating Client SDKs

Use OpenAPI Generator to create client libraries:

### Python

```bash
openapi-generator-cli generate \
  -i openapi.yaml \
  -g python \
  -o ./clients/python \
  --package-name stablerisk_client
```

### JavaScript/TypeScript

```bash
openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-fetch \
  -o ./clients/typescript
```

### Go

```bash
openapi-generator-cli generate \
  -i openapi.yaml \
  -g go \
  -o ./clients/go
```

## Postman Collection

Import the OpenAPI spec into Postman:

1. Open Postman
2. Click "Import"
3. Select "openapi.yaml"
4. Configure environment variables:
   - `base_url`: http://localhost:8080/api/v1
   - `access_token`: (obtain via login)

## Testing

Validate the OpenAPI specification:

```bash
# Using openapi-spec-validator
pip install openapi-spec-validator
openapi-spec-validator openapi.yaml

# Using swagger-cli
npm install -g @apidevtools/swagger-cli
swagger-cli validate openapi.yaml
```

## Compliance Notes

### ISO27001
- A.13.1 Network Security Management: TLS-only in production
- A.18.1.4 Protection of Records: Audit logging on all mutations

### PCI-DSS
- Requirement 6.5: Input validation documented in schemas
- Requirement 8: Authentication via JWT with short expiry
- Requirement 10: All API calls generate audit logs

## Support

For API questions or issues:
- GitHub Issues: https://github.com/yourusername/stablerisk/issues
- Email: support@stablerisk.example.com
