import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Routes, Route, MemoryRouter } from 'react-router-dom'
import { SiteDashboard } from '../SiteDashboard'
import { describe, it, expect, vi } from 'vitest'
import React from 'react'

// Mock the toast store
vi.mock('@/stores/toastStore', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn()
  }
}))

// Helper to render with MemoryRouter for route params
const renderWithParams = (projectId = '1', siteId = 'site-1') => {
  const queryClient = new QueryClient({
    defaultOptions: { 
      queries: { 
        retry: false,
        staleTime: 0
      } 
    }
  })

  return render(
    React.createElement(MemoryRouter, { initialEntries: [`/projects/${projectId}/sites/${siteId}`] },
      React.createElement(QueryClientProvider, { client: queryClient },
        React.createElement(Routes, null,
          React.createElement(Route, { 
            path: '/projects/:projectId/sites/:siteId', 
            element: React.createElement(SiteDashboard) 
          })
        )
      )
    )
  )
}

describe('SiteDashboard', () => {
  it('renders site information when loaded', async () => {
    renderWithParams()
    
    // Wait for site name to appear (appears in both breadcrumb and header)
    await waitFor(() => {
      const siteNames = screen.getAllByText('Test Site')
      expect(siteNames.length).toBeGreaterThanOrEqual(1)
    })
  })

  it('renders stats cards', async () => {
    renderWithParams()
    
    await waitFor(() => {
      expect(screen.getByText('Total VMs')).toBeInTheDocument()
      expect(screen.getByText('Running')).toBeInTheDocument()
      expect(screen.getByText('Total vCPUs')).toBeInTheDocument()
      expect(screen.getByText('Total Memory')).toBeInTheDocument()
    })
  })

  it('renders tab navigation', async () => {
    renderWithParams()
    
    await waitFor(() => {
      expect(screen.getByText('VMs')).toBeInTheDocument()
      expect(screen.getByText('Hosts')).toBeInTheDocument()
      expect(screen.getByText('Executions')).toBeInTheDocument()
    })
  })

  it('displays VM data in VMs tab', async () => {
    renderWithParams()
    
    await waitFor(() => {
      expect(screen.getByText('Test VM')).toBeInTheDocument()
    })
  })

  it('switches to hosts tab when clicked', async () => {
    renderWithParams()
    
    await waitFor(() => {
      expect(screen.getByText('Test VM')).toBeInTheDocument()
    })
    
    // Click on Hosts tab
    fireEvent.click(screen.getByText('Hosts'))
    
    await waitFor(() => {
      expect(screen.getByText('test-host')).toBeInTheDocument()
    })
  })

  it('opens create VM modal when button is clicked', async () => {
    renderWithParams()
    
    await waitFor(() => {
      expect(screen.getByText('Create VM')).toBeInTheDocument()
    })
    
    fireEvent.click(screen.getByText('Create VM'))
    
    // Modal should open - looking for the modal title
    await waitFor(() => {
      expect(screen.getByText('Create Virtual Machine')).toBeInTheDocument()
    })
  })

  it('displays correct VM counts in stats', async () => {
    renderWithParams()
    
    await waitFor(() => {
      // Check for Total VMs stat (1 from mock)
      const totalVms = screen.getByText('Total VMs').nextElementSibling
      expect(totalVms).toHaveTextContent('1')
    })
  })
})
