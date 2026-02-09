import { renderHook, waitFor } from '@testing-library/react'
import { useCreateTenant, useTenant } from '../../hooks.ts'
import { describe, it, expect, vi } from 'vitest'
import { createWrapper } from '../../../test/utils.js'

describe('useCreateTenant', () => {
  it('should create a tenant successfully', async () => {
    const onSuccess = vi.fn()
    const { result } = renderHook(
      () => useCreateTenant({
        onSuccess: (data, variables, context) => {
          onSuccess(data)
        }
      }),
      { wrapper: createWrapper() }
    )
    
    result.current.mutate({
      name: 'New Tenant',
      slug: 'new-tenant'
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data?.name).toBe('New Tenant')
    expect(result.current.data?.slug).toBe('new-tenant')
    expect(onSuccess).toHaveBeenCalled()
  })
})

describe('useTenant', () => {
  it('should fetch a single tenant', async () => {
    const { result } = renderHook(() => useTenant('1'), {
      wrapper: createWrapper()
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data?.id).toBe('1')
    expect(result.current.data?.name).toBe('Test Tenant')
  })

  it('should not fetch when tenantId is empty', () => {
    const { result } = renderHook(() => useTenant(''), {
      wrapper: createWrapper()
    })
    
    expect(result.current.isLoading).toBe(false)
    expect(result.current.fetchStatus).toBe('idle')
  })
})
