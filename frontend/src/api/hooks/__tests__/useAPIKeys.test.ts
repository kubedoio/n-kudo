import { renderHook, waitFor } from '@testing-library/react'
import { useAPIKeys, useCreateAPIKey, useRevokeAPIKey } from '../../hooks.ts'
import { describe, it, expect, vi } from 'vitest'
import { createWrapper } from '../../../test/utils.js'

describe('useAPIKeys', () => {
  it('should fetch API keys for a tenant', async () => {
    const { result } = renderHook(() => useAPIKeys('1'), {
      wrapper: createWrapper()
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0].name).toBe('Test API Key')
  })

  it('should not fetch when tenantId is empty', () => {
    const { result } = renderHook(() => useAPIKeys(''), {
      wrapper: createWrapper()
    })
    
    expect(result.current.isLoading).toBe(false)
    expect(result.current.fetchStatus).toBe('idle')
  })
})

describe('useCreateAPIKey', () => {
  it('should create an API key successfully', async () => {
    const onSuccess = vi.fn()
    const { result } = renderHook(
      () => useCreateAPIKey({
        onSuccess: (data) => {
          onSuccess(data)
        }
      }),
      { wrapper: createWrapper() }
    )
    
    result.current.mutate({
      tenantId: '1',
      data: { name: 'New API Key' }
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data?.name).toBe('New API Key')
    expect(result.current.data?.api_key).toBeDefined()
  })
})

describe('useRevokeAPIKey', () => {
  it('should revoke an API key successfully', async () => {
    const onSuccess = vi.fn()
    const { result } = renderHook(
      () => useRevokeAPIKey({
        onSuccess: (data) => {
          onSuccess(data)
        }
      }),
      { wrapper: createWrapper() }
    )
    
    result.current.mutate({
      tenantId: '1',
      keyId: 'key-1'
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
  })
})
