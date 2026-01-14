# Privé Dashboard

Modern React-based admin dashboard for Privé EDR/DLP Platform, inspired by Silva Admin Template.

## Features

### Core Dashboard Views

1. **Main Dashboard** (`/dashboard`)
   - Real-time event metrics (events/hour, critical alerts, active endpoints)
   - 24-hour event timeline with severity overlay
   - MITRE ATT&CK heat map
   - Severity distribution pie chart
   - Top affected endpoints
   - Recent critical alerts feed

2. **Threat Hunting** (`/hunting`)
   - SQL query builder with syntax highlighting
   - Process execution tree visualization
   - Event timeline analysis
   - IOC correlation matrix
   - Query results table with filters

3. **DLP Management** (`/dlp`)
   - Policy CRUD operations
   - Violation trends (30-day history)
   - Top violated policies ranking
   - Data classification breakdown
   - Recent violations feed
   - Policy effectiveness metrics

4. **License Management** (`/admin/licenses`)
   - License overview dashboard
   - Create/edit licenses
   - Usage tracking per license
   - Expiration warnings
   - Revenue analytics
   - Tier distribution

5. **Agent Management** (`/agents`)
   - Agent inventory
   - Health status monitoring
   - Configuration management
   - Deployment history
   - Performance metrics

6. **Compliance Reporting** (`/compliance`)
   - GDPR compliance status
   - HIPAA audit logs
   - SOC 2 controls mapping
   - ISO 27001 checkpoints
   - Export audit reports

## Tech Stack

- **Framework**: React 18 with hooks
- **Routing**: React Router v6
- **UI Library**: Material-UI (MUI) v5
- **Charts**: Recharts
- **State Management**: Zustand
- **HTTP Client**: Axios with React Query
- **Animations**: Framer Motion
- **Notifications**: React Hot Toast
- **Styling**: Emotion (CSS-in-JS)

## Getting Started

### Prerequisites

```bash
node >= 18.0.0
npm >= 9.0.0
```

### Installation

```bash
cd dashboard
npm install
```

### Development

```bash
# Start development server
npm start

# Opens http://localhost:3000
```

### Build for Production

```bash
npm run build

# Creates optimized build in /build
```

### Environment Variables

Create `.env` file:

```bash
REACT_APP_API_URL=http://localhost:8080
REACT_APP_WS_URL=ws://localhost:8080/ws
REACT_APP_VERSION=1.0.0
```

## Project Structure

```
dashboard/
├── public/
│   ├── index.html
│   ├── favicon.ico
│   └── manifest.json
├── src/
│   ├── components/
│   │   ├── layout/
│   │   │   ├── Sidebar.jsx
│   │   │   ├── Header.jsx
│   │   │   └── Footer.jsx
│   │   ├── dashboard/
│   │   │   ├── EventMetrics.jsx
│   │   │   ├── EventTimeline.jsx
│   │   │   ├── MITREHeatMap.jsx
│   │   │   └── AlertsFeed.jsx
│   │   ├── licenses/
│   │   │   ├── LicenseTable.jsx
│   │   │   ├── LicenseForm.jsx
│   │   │   ├── UsageChart.jsx
│   │   │   └── TierBadge.jsx
│   │   └── common/
│   │       ├── Card.jsx
│   │       ├── Chart.jsx
│   │       ├── Table.jsx
│   │       └── Button.jsx
│   ├── pages/
│   │   ├── Dashboard.jsx
│   │   ├── ThreatHunting.jsx
│   │   ├── DLPManagement.jsx
│   │   ├── LicenseManagement.jsx
│   │   ├── AgentManagement.jsx
│   │   └── Settings.jsx
│   ├── services/
│   │   ├── api.js
│   │   ├── authService.js
│   │   └── licenseService.js
│   ├── hooks/
│   │   ├── useAuth.js
│   │   ├── useLicenses.js
│   │   └── useWebSocket.js
│   ├── styles/
│   │   ├── theme.js
│   │   └── global.css
│   ├── App.jsx
│   └── index.jsx
├── package.json
└── README.md
```

## Key Components

### Dashboard Card
```jsx
<Card
  title="Critical Alerts"
  value="23"
  trend="-8 from yesterday"
  trendUp={false}
  icon={<WarningIcon />}
/>
```

### License Table
```jsx
<LicenseTable
  licenses={licenses}
  onRevoke={handleRevoke}
  onExtend={handleExtend}
/>
```

### Event Timeline
```jsx
<EventTimeline
  data={events}
  timeRange="24h"
  onPointClick={handleEventClick}
/>
```

## Styling

### Theme Configuration

The dashboard uses a dark theme with accent colors:

```js
const theme = {
  palette: {
    primary: '#667eea',    // Purple
    secondary: '#764ba2',  // Darker purple
    critical: '#ef4444',   // Red
    high: '#f59e0b',       // Orange
    medium: '#fbbf24',     // Yellow
    low: '#10b981',        // Green
    info: '#3b82f6',       // Blue
    bg: '#1f2937',         // Dark gray
    card: '#374151',       // Lighter gray
  }
}
```

### Responsive Design

All components are mobile-responsive:
- Desktop: >= 1280px (full layout)
- Tablet: 768px - 1279px (collapsible sidebar)
- Mobile: < 768px (bottom navigation)

## API Integration

### License API Example

```javascript
import { useLicenses } from './hooks/useLicenses';

function LicenseManagement() {
  const { licenses, createLicense, revokeLicense } = useLicenses();

  const handleCreate = async (data) => {
    await createLicense(data);
    toast.success('License created successfully');
  };

  return (
    <LicenseTable
      licenses={licenses}
      onCreate={handleCreate}
    />
  );
}
```

### Real-time Updates

```javascript
import { useWebSocket } from './hooks/useWebSocket';

function Dashboard() {
  const { data: events } = useWebSocket('/ws/events');

  return (
    <EventTimeline data={events} />
  );
}
```

## Authentication

Dashboard includes JWT-based authentication:

```javascript
// Login
const { login } = useAuth();
await login(email, password);

// Protected routes
<PrivateRoute path="/admin" component={AdminPanel} />
```

## Performance Optimization

- **Code Splitting**: Lazy load routes
- **Memoization**: React.memo for heavy components
- **Virtual Scrolling**: For large tables
- **Debouncing**: Search and filter inputs
- **Service Worker**: Offline support

## Deployment

### Nginx Configuration

```nginx
server {
    listen 80;
    server_name dashboard.prive-security.com;

    root /var/www/prive-dashboard/build;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_cache_bypass $http_upgrade;
    }
}
```

### Docker Deployment

```dockerfile
FROM node:18-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## Testing

```bash
# Run tests
npm test

# Coverage report
npm test -- --coverage

# E2E tests (Cypress)
npm run cypress:open
```

## Browser Support

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

Copyright © 2026 Privé Security. All rights reserved.

## Support

- Documentation: https://docs.prive-security.com
- Issues: https://github.com/prive-security/dashboard/issues
- Email: support@prive-security.com
