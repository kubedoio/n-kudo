# n-kudo MVP-1 UI — Developer Guide

## Overview

Operator console for the n-kudo control plane. Built with:

- **React 18** + **TypeScript**
- **Vite** (dev server + build)
- **TailwindCSS** (dark mode, CSS variable theming)
- **shadcn/ui** primitives (Radix UI underneath)
- **TanStack Query** (React Query v5)
- **React Router** v6

## Running

```bash
cd frontend
npm install
npm run dev          # → http://localhost:5173
npm run build        # production bundle
```

## Required Configuration

Open **Profile & Settings** (`/profile`) and fill in:

| Field | Description | Default |
|---|---|---|
| API Base URL | Control plane URL or `/api` for dev proxy | `/api` |
| Tenant ID | UUID of the tenant to operate on | — |
| Admin Key | `X-Admin-Key` for bootstrap endpoints | — |
| API Key | `X-API-Key` for tenant-scoped endpoints | — |

Values are stored in `localStorage` and used by the Axios HTTP client.

## API Endpoints Used

| Function | Method | Path |
|---|---|---|
| `healthz()` | GET | `/healthz` |
| `listSites(tenantId)` | GET | `/tenants/{tenantID}/sites` |
| `createSite(tenantId, payload)` | POST | `/tenants/{tenantID}/sites` |
| `createEnrollmentToken(tenantId, payload)` | POST | `/tenants/{tenantID}/enrollment-tokens` |
| `listHosts(siteId)` | GET | `/sites/{siteID}/hosts` |
| `listVMs(siteId)` | GET | `/sites/{siteID}/vms` |
| `createPlan(siteId, planPayload)` | POST | `/sites/{siteID}/plans` |
| `listExecutions(siteId)` | GET | `/sites/{siteID}/executions` |
| `getExecutionLogs(executionId)` | GET | `/executions/{executionID}/logs` |

Auth headers are injected automatically:
- `X-Admin-Key`: for admin endpoints (create tenant, etc.)
- `X-API-Key`: for tenant-scoped endpoints (sites, plans, VMs, etc.)

## Architecture

```
src/
├── api/              # HTTP client, types, hooks (no UI code)
│   ├── http.ts       # Axios instance with auth headers
│   ├── types.ts      # TypeScript interfaces
│   ├── client.ts     # API functions
│   ├── queryKeys.ts  # TanStack Query keys
│   └── hooks.ts      # React Query hooks
├── components/
│   ├── ui/           # shadcn/ui primitives
│   ├── AppShell.tsx  # Layout + header
│   ├── SitesPanel.tsx
│   ├── ActionsPanel.tsx
│   ├── LivePanel.tsx
│   ├── VMTable.tsx
│   ├── ExecutionList.tsx
│   └── ...
├── pages/            # Route-level components
│   ├── DashboardPage.tsx
│   ├── SitePage.tsx
│   ├── EnrollmentPage.tsx
│   ├── ExecutionPage.tsx
│   ├── NotificationsPage.tsx
│   └── ProfilePage.tsx
├── lib/
│   └── utils.ts      # cn(), formatRelativeTime(), etc.
├── App.tsx           # Router + QueryProvider
└── main.tsx          # Entry point
```

## Extending to MVP-2 Designer

The **Designer** button in the header is already present but disabled with an "MVP-2" badge. To enable it:

1. Create `src/pages/DesignerPage.tsx`
2. Add a route: `<Route path="/designer" element={<DesignerPage />} />`
3. In `AppShell.tsx`, replace the disabled button with a `<NavLink to="/designer">`
4. The designer canvas can use react-flow or similar for node-based infrastructure design
5. Template CRUD endpoints can be integrated via new functions in `src/api/client.ts`

## Extending Notifications

MVP-1 uses a placeholder. To enable real-time:
1. Add SSE client in `src/api/sse.ts`
2. Subscribe to execution/audit events
3. Show toast notifications using a Radix Toast primitive
