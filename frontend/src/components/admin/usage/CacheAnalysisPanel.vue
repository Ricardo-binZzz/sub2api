<template>
  <section class="card overflow-hidden" data-testid="cache-analysis-panel">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
      <div class="flex min-w-0 items-center gap-3">
        <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-cyan-50 text-cyan-700 dark:bg-cyan-950/40 dark:text-cyan-300">
          <Icon name="database" size="md" />
        </div>
        <div class="min-w-0">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('usage.cacheAnalysis.title') }}</h2>
          <p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ t('usage.cacheAnalysis.subtitle') }}</p>
        </div>
      </div>
      <span class="rounded-md px-2 py-1 text-xs font-medium" :class="statusClass(overallStatus)">
        {{ statusLabel(overallStatus) }}
      </span>
    </div>

    <div class="grid grid-cols-2 border-b border-gray-100 dark:border-dark-700 lg:grid-cols-4">
      <div v-for="metric in metrics" :key="metric.label" class="min-w-0 border-r border-gray-100 px-4 py-3 last:border-r-0 dark:border-dark-700">
        <p class="truncate text-xs text-gray-500 dark:text-gray-400">{{ metric.label }}</p>
        <p class="mt-1 text-lg font-semibold tabular-nums text-gray-900 dark:text-white">{{ metric.value }}</p>
      </div>
    </div>

    <div class="grid lg:grid-cols-[minmax(0,1fr)_minmax(300px,0.42fr)]">
      <div class="min-w-0 overflow-x-auto lg:border-r lg:border-gray-100 lg:dark:border-dark-700">
        <table class="w-full min-w-[620px]">
          <thead class="bg-gray-50/80 text-left text-xs font-medium text-gray-500 dark:bg-dark-800/60 dark:text-gray-400">
            <tr>
              <th class="px-4 py-2.5">{{ t('usage.model') }}</th>
              <th class="px-3 py-2.5 text-right">{{ t('usage.cacheAnalysis.promptTokens') }}</th>
              <th class="px-3 py-2.5 text-right">{{ t('usage.cacheRead') }}</th>
              <th class="px-3 py-2.5 text-right">{{ t('usage.cacheWrite') }}</th>
              <th class="px-3 py-2.5 text-right">{{ t('usage.cacheAnalysis.tokenHitRate') }}</th>
              <th class="px-4 py-2.5 text-right">{{ t('usage.cacheAnalysis.diagnosis') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-if="loading">
              <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</td>
            </tr>
            <tr v-else-if="modelRows.length === 0">
              <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('usage.cacheAnalysis.noData') }}</td>
            </tr>
            <tr v-for="row in modelRows" v-else :key="row.model" class="text-sm">
              <td class="max-w-64 truncate px-4 py-2.5 font-medium text-gray-800 dark:text-gray-200" :title="row.model">{{ row.model }}</td>
              <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-300">{{ formatTokens(row.promptTokens) }}</td>
              <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-300">{{ formatTokens(row.cacheReadTokens) }}</td>
              <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-300">{{ formatTokens(row.cacheCreationTokens) }}</td>
              <td class="px-3 py-2.5 text-right font-medium tabular-nums text-gray-800 dark:text-gray-200">{{ formatPercent(row.hitRate) }}</td>
              <td class="px-4 py-2.5 text-right">
                <span class="inline-flex rounded-md px-2 py-1 text-xs font-medium" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="flex min-h-40 flex-col justify-center px-4 py-4">
        <p class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('usage.cacheAnalysis.diagnosis') }}</p>
        <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ statusLabel(overallStatus) }}</p>
        <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ statusDescription(overallStatus) }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ModelStat } from '@/types'
import type { AdminUsageStatsResponse } from '@/api/admin/usage'

type CacheStatus = 'no_data' | 'inactive' | 'low' | 'churn' | 'fair' | 'good'

const props = withDefaults(defineProps<{
  stats: AdminUsageStatsResponse | null
  models: ModelStat[]
  loading?: boolean
}>(), {
  loading: false,
})

const { t } = useI18n()

const promptTokens = computed(() =>
  (props.stats?.total_input_tokens ?? 0)
  + (props.stats?.total_cache_creation_tokens ?? 0)
  + (props.stats?.total_cache_read_tokens ?? 0)
)
const cacheActivityTokens = computed(() =>
  (props.stats?.total_cache_creation_tokens ?? 0) + (props.stats?.total_cache_read_tokens ?? 0)
)
const cacheHitRate = computed(() => ratio(props.stats?.total_cache_read_tokens ?? 0, promptTokens.value))
const cacheReuseRatio = computed(() => ratio(props.stats?.total_cache_read_tokens ?? 0, cacheActivityTokens.value))
const overallStatus = computed(() => classify(
  promptTokens.value,
  props.stats?.total_cache_creation_tokens ?? 0,
  props.stats?.total_cache_read_tokens ?? 0,
))

const metrics = computed(() => [
  { label: t('usage.cacheAnalysis.tokenHitRate'), value: formatPercent(cacheHitRate.value) },
  { label: t('usage.cacheAnalysis.reuseRatio'), value: formatPercent(cacheReuseRatio.value) },
  { label: t('usage.cacheAnalysis.cacheReadVolume'), value: formatTokens(props.stats?.total_cache_read_tokens ?? 0) },
  { label: t('usage.cacheAnalysis.cacheWriteVolume'), value: formatTokens(props.stats?.total_cache_creation_tokens ?? 0) },
])

const modelRows = computed(() => props.models
  .map((model) => {
    const modelPromptTokens = model.input_tokens + model.cache_creation_tokens + model.cache_read_tokens
    return {
      model: model.model,
      promptTokens: modelPromptTokens,
      cacheCreationTokens: model.cache_creation_tokens,
      cacheReadTokens: model.cache_read_tokens,
      hitRate: ratio(model.cache_read_tokens, modelPromptTokens),
      status: classify(modelPromptTokens, model.cache_creation_tokens, model.cache_read_tokens),
    }
  })
  .filter((row) => row.promptTokens > 0)
  .sort((a, b) => b.promptTokens - a.promptTokens)
  .slice(0, 6))

function ratio(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0
}

function classify(totalPromptTokens: number, cacheCreationTokens: number, cacheReadTokens: number): CacheStatus {
  if (totalPromptTokens <= 0) return 'no_data'
  if (cacheCreationTokens + cacheReadTokens <= 0) return 'inactive'
  const hitRate = ratio(cacheReadTokens, totalPromptTokens)
  if (hitRate >= 0.5) return 'good'
  if (hitRate >= 0.2) return 'fair'
  if (cacheCreationTokens > cacheReadTokens) return 'churn'
  return 'low'
}

function statusLabel(status: CacheStatus): string {
  return t(`usage.cacheAnalysis.status.${status}`)
}

function statusDescription(status: CacheStatus): string {
  return t(`usage.cacheAnalysis.description.${status}`)
}

function statusClass(status: CacheStatus): string {
  if (status === 'good') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
  if (status === 'fair') return 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
  if (status === 'churn') return 'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
  if (status === 'low') return 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(1)}%`
}

function formatTokens(value: number): string {
  if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`
  if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`
  if (value >= 1e3) return `${(value / 1e3).toFixed(2)}K`
  return value.toLocaleString()
}
</script>
