import { renderHook, waitFor } from '@testing-library/react'
import { useTenants } from '../../hooks.ts'
import { describe, it, expect } from 'vitest'
import { createWrapper } from '../../../test/utils.js'

describe('useTenants', () => {
  it('should fetch tenants successfully', async () => {
    const { result } = renderHook(() => useTenants(), {
      wrapper: createWrapper()
    })
    
    // Wait for loading to complete (avoids race condition)
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    
    // Then check success
    expect(result.current.isSuccess).toBe(true)
    expect(result.current.data).toBeDefined()
    expect(result.current.data).toHaveLength(2)
    expect(result.current.data?.[0].name).toBe('Test Tenant')
  })
})
