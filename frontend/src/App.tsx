import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { ErrorBoundary } from './components/ErrorBoundary'
import { TenantsList } from './pages/Admin/TenantsList'
import { TenantDetail } from './pages/Admin/TenantDetail'
import { SitesList, SiteDashboard } from './pages/Tenant'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/admin/tenants" element={
          <ErrorBoundary><TenantsList /></ErrorBoundary>
        } />
        <Route path="/admin/tenants/:tenantId" element={
          <ErrorBoundary><TenantDetail /></ErrorBoundary>
        } />
        <Route path="/tenant/:tenantId/sites" element={
          <ErrorBoundary><SitesList /></ErrorBoundary>
        } />
        <Route path="/tenant/:tenantId/sites/:siteId" element={
          <ErrorBoundary><SiteDashboard /></ErrorBoundary>
        } />
        <Route path="/" element={<Navigate to="/admin/tenants" replace />} />
        {/* Fallback for unmatched routes */}
        <Route path="*" element={<Navigate to="/admin/tenants" replace />} />
      </Routes>
    </Layout>
  )
}

export default App
