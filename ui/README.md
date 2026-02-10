# n-kudo MVP-1 UI

A modern, minimal cloud control-plane dashboard for managing "sub-clouds" (sites, hosts, and MicroVMs).

## Features
- **Multi-tenant**: Secure connection using Tenant ID and API Key.
- **Site Management**: Onboard new sites and view connectivity status.
- **Host Discovery**: View hardware facts and resource utilization of edge hosts.
- **MicroVM Lifecycle**: Create and start VMs using Cloud Hypervisor templates.
- **Real-time Logs**: Stream execution logs via polling for plan status and debugging.
- **Secure**: API keys are stored in-memory or session-only.

## Tech Stack
- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Data Fetching**: TanStack Query (React Query)
- **Icons**: Lucide React

## Getting Started

### Prerequisites
- Node.js 18+
- A running n-kudo control-plane

### Installation
```bash
cd ui
npm install
```

### Development
```bash
npm run dev
```
The UI will be available at [http://localhost:3000](http://localhost:3000).

### Environment Configuration
The UI prompts for the Control-plane URL on the connection page. For development against self-signed certificates:
```bash
export NODE_TLS_REJECT_UNAUTHORIZED=0
npm run dev
```

## Dashboard Layout
- **Sites**: Overview of all managed edge locations.
- **Hosts**: Inventory of hardware nodes in a site.
- **VMs**: List of virtual machines and their current state.
- **Enrollment**: One-time token generation for onboarding new agents.
- **Executions**: History and live logs of plan applications.

## Design Notes
- **Polling over WebSockets**: Chosen for MVP-1 reliability and simpler backend implementation.
- **Session Persistence**: API keys are never stored in `localStorage` to prevent persistent credential leaks.
- **Responsive Tables**: Designed for high-density information display typical of cloud consoles.
