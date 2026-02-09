import { renderHook, waitFor } from '@testing-library/react'
import { useSites, useSite } from '../../hooks.ts'
import { describe, it, expect } from 'vitest'
import { createWrapper } from '../../../test/utils.js'

describe('useSites', () => {
  it('should fetch sites for a tenant', async () => {
    const { result } = renderHook(() => useSites('1'), {
      wrapper: createWrapper()
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0].name).toBe('Test Site')
    expect(result.current.data?.[0].connectivity_state).toBe('ONLINE')
  })

  it('should not fetch when tenantId is empty', () => {
    const { result } = renderHook(() => useSites(''), {
      wrapper: createWrapper()
    })
    
    expect(result.current.isLoading).toBe(false)
    expect(result.current.fetchStatus).toBe('idle')
  })
})

describe('useSite', () => {
  it('should fetch a specific site', async () => {
    const { result } = renderHook(() => useSite('1', 'site-1'), {
      wrapper: createWrapper()
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data?.name).toBe('Test Site')
  })

  it('should not fetch when tenantId or siteId is empty', () => {
    const { result } = renderHook(() => useSite('', 'site-1'), {
      wrapper: createWrapper()
    })
    
    expect(result.current.isLoading).toBe(false)
    expect(result.current.fetchStatus).toBe('idle')
  })
})
