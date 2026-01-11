# StableRisk Web Dashboard

Modern web dashboard for StableRisk USDT transaction monitoring system built with SvelteKit, TailwindCSS, and DaisyUI.

## Features

- **Real-time Updates**: WebSocket integration for live outlier notifications
- **Authentication**: JWT-based authentication with role-based access control
- **Responsive Design**: Mobile-friendly interface with dark mode support
- **Outlier Management**: Browse, filter, and acknowledge detected anomalies
- **Statistics Dashboard**: Comprehensive analytics and trend visualization
- **Graph Visualization**: D3.js-powered transaction network visualization

## Tech Stack

- **Framework**: SvelteKit 2.x
- **UI**: TailwindCSS 3.x + DaisyUI 4.x
- **Visualization**: D3.js 7.x, Chart.js 4.x
- **Language**: TypeScript
- **Build**: Vite
- **Deployment**: Node.js adapter

## Development

### Prerequisites

- Node.js 20+
- npm or pnpm

### Installation

```bash
npm install
```

### Configuration

The dashboard expects the API to be available at `/api/v1`. In development, Vite proxies requests to the backend:

```javascript
// vite.config.js
server: {
  proxy: {
    '/api': {
      target: process.env.API_URL || 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

### Running

```bash
# Development server (port 3000)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking
npm run check
```

## Project Structure

```
web/
├── src/
│   ├── lib/
│   │   ├── api/              # API client and types
│   │   │   ├── client.ts     # HTTP client
│   │   │   └── types.ts      # TypeScript interfaces
│   │   ├── stores/           # Svelte stores
│   │   │   ├── auth.ts       # Authentication state
│   │   │   └── websocket.ts  # WebSocket connection
│   │   └── components/       # Reusable components
│   │       └── GraphVisualization.svelte
│   ├── routes/               # Pages
│   │   ├── +layout.svelte    # Main layout
│   │   ├── +page.svelte      # Dashboard
│   │   ├── login/            # Login page
│   │   ├── outliers/         # Outliers page
│   │   └── statistics/       # Statistics page
│   ├── app.css               # Global styles
│   └── app.html              # HTML template
├── static/                   # Static assets
├── Dockerfile                # Production container
├── package.json
├── svelte.config.js
├── tailwind.config.js
└── vite.config.js
```

## API Integration

### Authentication

```typescript
import { auth } from '$stores/auth';

// Login
await auth.login('username', 'password');

// Logout
auth.logout();

// Access user
$auth.user
```

### API Client

```typescript
import apiClient from '$api/client';

// List outliers
const outliers = await apiClient.listOutliers({
  page: 1,
  limit: 20,
  severity: 'high'
});

// Get statistics
const stats = await apiClient.getStatistics();

// Acknowledge outlier
await apiClient.acknowledgeOutlier(id, { notes: '...' });
```

### WebSocket

```typescript
import { websocket, outlierMessages } from '$stores/websocket';

// Subscribe to specific types/severities
websocket.setFilters({
  severities: ['high', 'critical'],
  types: ['zscore']
});

// React to new outliers
$: if ($outlierMessages) {
  console.log('New outlier:', $outlierMessages);
}
```

## Roles & Permissions

### Admin
- Full access to all features
- User management
- System configuration

### Analyst
- View outliers, transactions, statistics
- Acknowledge outliers
- Trigger manual detection

### Viewer
- Read-only access
- View outliers, transactions, statistics
- No modification permissions

## Default Credentials

For development/testing:

- **Admin**: admin / changeme123
- **Analyst**: analyst / changeme123
- **Viewer**: viewer / changeme123

⚠️ **Change these in production!**

## Docker Deployment

```bash
# Build image
docker build -t stablerisk-web .

# Run container
docker run -p 3000:3000 \
  -e API_URL=http://api:8080 \
  stablerisk-web
```

## Production Considerations

1. **Environment Variables**:
   - `NODE_ENV=production`
   - `API_URL`: Backend API URL
   - `PORT`: Server port (default: 3000)

2. **HTTPS**: Use a reverse proxy (Nginx) for TLS termination

3. **API Proxy**: Configure proper CORS or use same-origin deployment

4. **Build Optimization**: SvelteKit automatically optimizes for production

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari 14+, Chrome Mobile)

## License

Copyright © 2025 StableRisk
