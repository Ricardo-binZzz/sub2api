import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CacheAnalysisPanel from '../CacheAnalysisPanel.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const stats = {
  total_requests: 4,
  total_input_tokens: 200,
  total_output_tokens: 50,
  total_cache_tokens: 400,
  total_cache_creation_tokens: 100,
  total_cache_read_tokens: 300,
  total_tokens: 650,
  total_cost: 1,
  total_actual_cost: 0.8,
  total_account_cost: 0.6,
  average_duration_ms: 100,
}

describe('CacheAnalysisPanel', () => {
  it('calculates token hit rate and model diagnosis from existing aggregates', () => {
    const wrapper = mount(CacheAnalysisPanel, {
      props: {
        stats,
        models: [{
          model: 'gpt-cache',
          requests: 2,
          input_tokens: 100,
          output_tokens: 10,
          cache_creation_tokens: 50,
          cache_read_tokens: 150,
          total_tokens: 310,
          cost: 0.5,
          actual_cost: 0.4,
        }],
      },
      global: { stubs: { Icon: true } },
    })

    const text = wrapper.text()
    expect(text).toContain('50.0%')
    expect(text).toContain('75.0%')
    expect(text).toContain('gpt-cache')
    expect(text).toContain('usage.cacheAnalysis.status.good')
  })

  it('flags cache creation without reuse as churn', () => {
    const wrapper = mount(CacheAnalysisPanel, {
      props: {
        stats: { ...stats, total_input_tokens: 500, total_cache_creation_tokens: 200, total_cache_read_tokens: 10 },
        models: [],
      },
      global: { stubs: { Icon: true } },
    })

    expect(wrapper.text()).toContain('usage.cacheAnalysis.status.churn')
  })
})
