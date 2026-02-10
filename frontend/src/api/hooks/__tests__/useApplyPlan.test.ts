import { renderHook, waitFor } from '@testing-library/react'
import { useApplyPlan, useApplyPlanFromActions } from '../../hooks.ts'
import { describe, it, expect, vi } from 'vitest'
import { createWrapper } from '../../../test/utils.js'

describe('useApplyPlan', () => {
  it('should apply a plan successfully', async () => {
    const onSuccess = vi.fn()
    const { result } = renderHook(
      () => useApplyPlan({
        onSuccess: (data) => {
          onSuccess(data)
        }
      }),
      { wrapper: createWrapper() }
    )
    
    result.current.mutate({
      siteId: 'site-1',
      plan: {
        idempotency_key: 'test-key-123',
        actions: [
          { operation: 'CREATE', name: 'New VM', vcpu_count: 2, memory_mib: 4096 }
        ]
      }
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data?.plan_id).toBe('plan-1')
    expect(result.current.data?.plan_version).toBe(1)
    expect(onSuccess).toHaveBeenCalled()
  })
})

describe('useApplyPlanFromActions', () => {
  it('should apply plan from actions successfully', async () => {
    const onSuccess = vi.fn()
    const { result } = renderHook(
      () => useApplyPlanFromActions({
        onSuccess: (data) => {
          onSuccess(data)
        }
      }),
      { wrapper: createWrapper() }
    )
    
    result.current.mutate({
      siteId: 'site-1',
      idempotencyKey: 'action-key-456',
      actions: [
        { operation: 'START', vm_id: 'vm-1' }
      ]
    })
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    
    expect(result.current.data).toBeDefined()
    expect(result.current.data?.plan_id).toBe('plan-1')
    expect(onSuccess).toHaveBeenCalled()
  })
})
