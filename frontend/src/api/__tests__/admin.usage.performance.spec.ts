import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '@/api/client'
import { getPerformance } from '@/api/admin/usage'

vi.mock('@/api/client', () => ({ apiClient: { get: vi.fn() } }))

describe('admin usage performance API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('requests the selected read-only aggregation period', async () => {
    const report = { groups: [], accounts: [], thresholds: {} }
    vi.mocked(apiClient.get).mockResolvedValue({ data: report })
    await expect(getPerformance('7d')).resolves.toBe(report)
    expect(apiClient.get).toHaveBeenCalledWith('/admin/usage/performance', { params: { period: '7d' } })
  })
})
