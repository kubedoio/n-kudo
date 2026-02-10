import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ProjectsList, ProjectDetail, ProjectSettings } from './pages/Projects';
import { SitesList, SiteDashboard } from './pages/Tenant';
import { Login, Register, VerifyEmail } from './pages/Auth';
import { isAuthenticated } from './api/auth';

// Protected route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

// Public route wrapper (redirects to dashboard if already authenticated)
function PublicRoute({ children }: { children: React.ReactNode }) {
  if (isAuthenticated()) {
    return <Navigate to="/projects" replace />
  }
  return <>{children}</>
}

function App() {
  return (
    <Routes>
      {/* Public auth routes (no layout) */}
      <Route path="/login" element={
        <PublicRoute>
          <Login />
        </PublicRoute>
      } />
      <Route path="/register" element={
        <PublicRoute>
          <Register />
        </PublicRoute>
      } />
      
      {/* Email verification (public) */}
      <Route path="/verify-email" element={<VerifyEmail />} />

      {/* Protected routes with layout */}
      <Route path="/" element={
        <ProtectedRoute>
          <Layout>
            <Navigate to="/projects" replace />
          </Layout>
        </ProtectedRoute>
      } />
      
      {/* Projects - Main entry for authenticated users */}
      <Route path="/projects" element={
        <ProtectedRoute>
          <Layout>
            <ErrorBoundary><ProjectsList /></ErrorBoundary>
          </Layout>
        </ProtectedRoute>
      } />
      
      {/* Project Detail */}
      <Route path="/projects/:projectId" element={
        <ProtectedRoute>
          <Layout>
            <ErrorBoundary><ProjectDetail /></ErrorBoundary>
          </Layout>
        </ProtectedRoute>
      } />
      
      {/* Project Settings */}
      <Route path="/projects/:projectId/settings" element={
        <ProtectedRoute>
          <Layout>
            <ErrorBoundary><ProjectSettings /></ErrorBoundary>
          </Layout>
        </ProtectedRoute>
      } />
      
      {/* Project Sites */}
      <Route path="/projects/:projectId/sites" element={
        <ProtectedRoute>
          <Layout>
            <ErrorBoundary><SitesList /></ErrorBoundary>
          </Layout>
        </ProtectedRoute>
      } />
      
      {/* Site Dashboard */}
      <Route path="/projects/:projectId/sites/:siteId" element={
        <ProtectedRoute>
          <Layout>
            <ErrorBoundary><SiteDashboard /></ErrorBoundary>
          </Layout>
        </ProtectedRoute>
      } />

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App
