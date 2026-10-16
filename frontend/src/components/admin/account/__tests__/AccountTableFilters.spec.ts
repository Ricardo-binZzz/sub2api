import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import AccountTableFilters from '../AccountTableFilters.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import type { AdminGroup } from '@/types'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
afterEach(() => vi.useRealTimers())

describe('merged flat account filters', () => {
  const filters = { platform: 'openai', type: 'cpr', status: 'active', privacy_mode: '__unset__', group: '42' }
  const render = () => mount(AccountTableFilters, { props: { searchQuery: 'existing', filters, groups: [{ id: 42, name: 'Custom group' } as AdminGroup] } })

  it('retains CPR and every filter without a collapsed panel', async () => {
    const wrapper = render()
    const selects = wrapper.findAllComponents(Select)
    expect(selects).toHaveLength(5)
    expect(wrapper.find('button[aria-controls]').exists()).toBe(false)
    const expected = [
      ['platform', 'openai'], ['type', 'cpr'], ['status', 'active'],
      ['privacy_mode', '__unset__'], ['group', '42'],
    ] as const
    for (const [i, [key, selected]] of expected.entries()) {
      expect(selects[i].props('modelValue')).toBe(selected)
      expect(selects[i].props('options').some((o: { value: string }) => o.value === selected)).toBe(true)
      selects[i].vm.$emit('update:modelValue', '')
      selects[i].vm.$emit('change')
      expect(wrapper.emitted('update:filters')?.at(-1)).toEqual([{ ...filters, [key]: '' }])
    }
    expect(wrapper.emitted('change')).toHaveLength(5)
    expect(wrapper.props('filters')).toEqual(filters)
  })

  it('retains immediate search updates and debounced search events', async () => {
    vi.useFakeTimers()
    const wrapper = render()
    await wrapper.getComponent(SearchInput).get('input').setValue('new search')
    expect(wrapper.emitted('update:searchQuery')).toEqual([['new search']])
    expect(wrapper.emitted('change')).toBeUndefined()
    await vi.advanceTimersByTimeAsync(300)
    expect(wrapper.emitted('change')).toEqual([[]])
  })
})
