# n-kudo Frontend

A modern React + TypeScript dashboard for the n-kudo MVP-1 control-plane.

## 🚀 Quick Start

```bash
# Install dependencies
npm install

# Start development server (connects to control-plane at :8443)
npm run dev

# Open http://localhost:3000
```

## 📁 Project Structure

```
frontend/
├── src/
│   ├── api/              # API client and TanStack Query hooks
│   │   ├── types.ts      # TypeScript interfaces
│   │   ├── client.ts     # Axios configuration
│   │   ├── api.ts        # API functions
│   │   ├── hooks.ts      # React Query hooks
│   │   └── index.ts      # Barrel exports
│   ├── components/
│   │   ├── common/       # Reusable UI components
│   │   ├── Layout.tsx    # Main layout wrapper
│   │   ├── Sidebar.tsx   # Navigation sidebar
│   │   └── Header.tsx    # Top header
│   ├── pages/
│   │   ├── Admin/        # Admin dashboard pages
│   │   │   ├── TenantsList.tsx
│   │   │   ├── TenantDetail.tsx
│   │   │   ├── CreateTenantModal.tsx
│   │   │   └── IssueTokenModal.tsx
│   │   └── Tenant/       # Operator dashboard pages
│   │       ├── SitesList.tsx
│   │       ├── SiteDashboard.tsx
│   │       ├── VMCreateModal.tsx
│   │       ├── VMActionsMenu.tsx
│   │       └── ExecutionLogViewer.tsx
│   ├── stores/           # Zustand state stores
│   └── utils/            # Utility functions
├── e2e/                  # Playwright E2E tests
└── public/               # Static assets
```

## 🛠️ Tech Stack

| Technology | Purpose |
|------------|---------|
| React 18 | UI library |
| TypeScript | Type safety |
| Vite | Build tool |
| Tailwind CSS | Styling |
| TanStack Query | Server state management |
| Zustand | Client state management |
| Axios | HTTP client |
| React Router | Navigation |
| Playwright | E2E testing |

## 📱 Features

### Admin Dashboard (`/admin/tenants`)
- ✅ Create tenants with auto-generated API keys
- ✅ Issue enrollment tokens for edge agents
- ✅ View all sites and their status
- ✅ Copy-to-clipboard for credentials

### Operator Dashboard (`/tenant/:tenantId/sites`)
- ✅ Site management with ONLINE/OFFLINE indicators
- ✅ VM lifecycle management (Create, Start, Stop, Delete)
- ✅ Real-time execution log streaming
- ✅ Host inventory with capability badges (KVM, Cloud Hypervisor)
- ✅ Plan history and status tracking

### UI Components
- ✅ Button, Card, Input, Select, Modal, Badge
- ✅ Table with sorting
- ✅ Toast notifications
- ✅ Loading states and skeletons
- ✅ Empty states

## 🔌 API Integration

The frontend connects to the n-kudo control-plane API:

```typescript
import { apiKeyStorage, useSites, useApplyPlan } from '@/api';

// Set API key after login
apiKeyStorage.setApiKey('nk_...');

// Use hooks in components
const { data: sites, isLoading } = useSites(tenantId);
const applyPlan = useApplyPlan();
```

### Auth Flow
1. Admin logs in with `X-Admin-Key` header
2. Creates tenant → receives `X-API-Key`
3. API key stored in localStorage
4. All subsequent requests use `X-API-Key` header

## 🧪 Testing

### Unit Tests
```bash
# Run unit tests
npm test
```

### E2E Tests
```bash
# Install Playwright browsers
npx playwright install

# Run E2E tests
npm run test:e2e

# Run with UI mode for debugging
npm run test:e2e:ui
```

### Test Structure
- `e2e/tenants.spec.ts` - Tenant management
- `e2e/sites.spec.ts` - Site management
- `e2e/vms.spec.ts` - VM lifecycle
- `e2e/plans.spec.ts` - Plan execution

## 📝 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `https://localhost:8443` | Control-plane API URL |
| `VITE_ADMIN_KEY` | `dev-admin-key` | Default admin key (dev only) |

Create `.env.local` to override:
```bash
VITE_API_BASE_URL=https://api.nkudo.io
```

## 🎨 Design System

### Colors
- Primary: `slate-900` (dark sidebar)
- Accent: `blue-600` (actions, links)
- Success: `green-500`
- Warning: `amber-500`
- Error: `red-500`

### Status Badges
- `PENDING` - Yellow
- `RUNNING` / `ONLINE` - Green
- `STOPPED` / `OFFLINE` - Gray
- `FAILED` - Red
- `SUCCEEDED` - Green

## 🚢 Deployment

### Build for Production
```bash
npm run build
```

Output goes to `dist/` directory.

### Docker
```bash
docker build -t nkudo-frontend .
docker run -p 80:80 nkudo-frontend
```

### Static Hosting
The built app is static and can be hosted on:
- Vercel
- Netlify
- GitHub Pages
- S3 + CloudFront
- Any CDN

## 🔄 Regenerating API Client

If the backend API changes, regenerate the TypeScript client:

```bash
npm run generate-api
```

This creates a fresh client from `../api/openapi.yaml`.

## 🐛 Troubleshooting

### CORS Errors
Ensure the control-plane allows requests from `http://localhost:3000`:
```bash
# In control-plane env
CONTROL_PLANE_CORS_ORIGINS=http://localhost:3000
```

### API Connection Failed
Check that the control-plane is running:
```bash
curl https://localhost:8443/healthz
```

### Type Errors After API Generation
Restart TypeScript service or IDE after running `generate-api`.

## 📚 Documentation

- [Backend README](../README.md)
- [API OpenAPI Spec](../api/openapi.yaml)
- [MVP-1 Architecture](../docs/mvp1/architecture.md)

## 🤝 Contributing

1. Create feature branch
2. Run tests: `npm run test:e2e`
3. Submit PR

## 📄 License

Same as n-kudo backend project.
