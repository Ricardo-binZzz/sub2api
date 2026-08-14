import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get }
}))

import { getGroupUpstreamRates } from '@/api/admin/groups'

describe('admin group upstream rates API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('requests the latest upstream declarations for a group', async () => {
    const rates = {
      group_id: 42,
      accounts: [{
        account_id: 1,
        account_name: 'Grok',
        platform: 'grok',
        status: 'active',
        schedulable: true,
        probe_supported: true,
        probe_status: 'ok',
        declared_rate_multiplier: 0.13,
        effective_rate_multiplier: 0.13,
        peak_rate_enabled: false,
        stale: false
      }]
    }
    get.mockResolvedValue({ data: rates })

    await expect(getGroupUpstreamRates(42)).resolves.toEqual(rates)
    expect(get).toHaveBeenCalledWith('/admin/groups/42/upstream-rates')
  })
})
