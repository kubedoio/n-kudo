import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { TenantsList } from '../TenantsList'
import { describe, it, expect, vi } from 'vitest'
import type { ReactNode } from 'react'
import React from 'react'

// Mock the toast store
vi.mock('@/stores/toastStore', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn()
  }
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { 
      queries: { 
        retry: false,
        staleTime: 0
      } 
    }
  })
  
  return ({ children }: { children: ReactNode }) =>
    React.createElement(BrowserRouter, null,
      React.createElement(QueryClientProvider, { client: queryClient }, children)
    )
}

describe('TenantsList', () => {
  it('renders the page header', () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    expect(screen.getByText('Tenants')).toBeInTheDocument()
    expect(screen.getByText('Manage organization tenants and their configurations')).toBeInTheDocument()
  })

  it('renders the create tenant button', () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    expect(screen.getByText('Create Tenant')).toBeInTheDocument()
  })

  it('renders search input', () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    expect(screen.getByPlaceholderText('Search tenants...')).toBeInTheDocument()
  })

  it('renders tenant data after loading', async () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    // Wait for the data to load
    await waitFor(() => {
      expect(screen.getByText('Test Tenant')).toBeInTheDocument()
    })
    
    // Use getAllByText since 'test' appears in both slug and region columns
    expect(screen.getAllByText('test').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('us-east-1')).toBeInTheDocument()
  })

  it('filters tenants when searching', async () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    // Wait for the data to load
    await waitFor(() => {
      expect(screen.getByText('Test Tenant')).toBeInTheDocument()
    })
    
    // Type in search box
    const searchInput = screen.getByPlaceholderText('Search tenants...')
    fireEvent.change(searchInput, { target: { value: 'Another' } })
    
    // Check that only matching tenant is shown
    await waitFor(() => {
      expect(screen.getByText('Another Tenant')).toBeInTheDocument()
      expect(screen.queryByText('Test Tenant')).not.toBeInTheDocument()
    })
  })

  it('opens create modal when create button is clicked', async () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    const createButton = screen.getByText('Create Tenant')
    fireEvent.click(createButton)
    
    // Check that modal is opened (looking for modal title)
    await waitFor(() => {
      expect(screen.getByText('Create New Tenant')).toBeInTheDocument()
    })
  })

  it('displays correct tenant count', async () => {
    render(React.createElement(TenantsList), { wrapper: createWrapper() })
    
    await waitFor(() => {
      expect(screen.getByText(/2 tenants/)).toBeInTheDocument()
    })
  })
})
