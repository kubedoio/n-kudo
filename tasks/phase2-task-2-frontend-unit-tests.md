# Phase 2 Task 2: Frontend Unit & Component Tests

## Task Description
Setup React Testing Library and implement unit tests for API hooks and components.

## Prerequisites
- Phase 1 complete
- Frontend build working

## Acceptance Criteria
- [ ] React Testing Library setup complete
- [ ] Unit tests for all API hooks
- [ ] Component tests for critical UI (TenantList, SiteDashboard)
- [ ] Tests run with `npm test`
- [ ] Coverage report generated

## Test Setup

### 1. Install Dependencies
```bash
cd frontend
npm install --save-dev @testing-library/react @testing-library/jest-dom @testing-library/user-event vitest @vitest/coverage-v8 msw
```

### 2. Configure Vitest
Create/update `frontend/vitest.config.ts`:
```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
    coverage: {
      reporter: ['text', 'json', 'html'],
      exclude: ['node_modules/', 'src/test/']
    }
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src')
    }
  }
})
```

### 3. Create Test Setup
Create `frontend/src/test/setup.ts`:
```typescript
import '@testing-library/jest-dom'
import { server } from './mocks/server'

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
```

### 4. Setup MSW (Mock Service Worker)
Create `frontend/src/test/mocks/handlers.ts`:
```typescript
import { http, HttpResponse } from 'msw'

export const handlers = [
  http.get('/tenants', () => {
    return HttpResponse.json([
      { id: '1', name: 'Test Tenant', slug: 'test', primary_region: 'us-east-1', data_retention_days: 30, created_at: '2024-01-01' }
    ])
  }),
  // Add more handlers...
]
```

Create `frontend/src/test/mocks/server.ts`:
```typescript
import { setupServer } from 'msw/node'
import { handlers } from './handlers'

export const server = setupServer(...handlers)
```

## Test Files to Create

### API Hook Tests

**`frontend/src/api/hooks/__tests__/useTenants.test.ts`**:
```typescript
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useTenants } from '../useTenants'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } }
})

const wrapper = ({ children }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
)

describe('useTenants', () => {
  it('fetches tenants successfully', async () => {
    const { result } = renderHook(() => useTenants(), { wrapper })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data[0].name).toBe('Test Tenant')
  })
})
```

Create similar tests for:
- `useTenant.test.ts`
- `useSites.test.ts`
- `useAPIKeys.test.ts`
- `useCreateTenant.test.ts`
- `useApplyPlan.test.ts`

### Component Tests

**`frontend/src/pages/Admin/__tests__/TenantsList.test.tsx`**:
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { TenantsList } from '../TenantsList'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const queryClient = new QueryClient()

const renderWithProviders = (component) => {
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>
  )
}

describe('TenantsList', () => {
  it('renders tenant list', async () => {
    renderWithProviders(<TenantsList />)
    
    await waitFor(() => {
      expect(screen.getByText('Test Tenant')).toBeInTheDocument()
    })
  })

  it('opens create modal when button clicked', async () => {
    renderWithProviders(<TenantsList />)
    
    fireEvent.click(screen.getByText('Create Tenant'))
    
    expect(screen.getByText('Create New Tenant')).toBeInTheDocument()
  })
})
```

Create similar tests for:
- `SiteDashboard.test.tsx` - Tests VM list, actions
- `TenantDetail.test.tsx` - Tests tabs, API keys

## Test Coverage Goals

| Category | Target Coverage |
|----------|-----------------|
| API Hooks | >90% |
| Components | >70% |
| Utils | >80% |

## Running Tests

```bash
cd /srv/data01/kubedo/n-kudo/frontend

# Run tests
npm test

# Run with coverage
npm test -- --coverage

# Run in watch mode
npm test -- --watch
```

## Definition of Done
- [ ] Testing libraries installed and configured
- [ ] MSW setup for API mocking
- [ ] Unit tests for all API hooks
- [ ] Component tests for critical UI
- [ ] Coverage report generated
- [ ] Tests pass in CI

## Estimated Effort
8-10 hours
