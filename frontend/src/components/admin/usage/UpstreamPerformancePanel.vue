<template>
  <section class="space-y-4 p-4 sm:p-5">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('usage.performance.title') }}</h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('usage.performance.description') }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex border border-gray-200 p-0.5 dark:border-dark-600">
          <button
            v-for="option in periods"
            :key="option.value"
            type="button"
            class="px-3 py-1.5 text-xs font-medium transition-colors"
            :class="period === option.value ? 'bg-primary-500 text-white' : 'text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-700'"
            @click="selectPeriod(option.value)"
          >
            {{ option.label }}
          </button>
        </div>
        <button
          type="button"
          class="btn btn-secondary px-2.5"
          :title="t('common.refresh')"
          :disabled="loading"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          <span class="hidden sm:inline">{{ t('common.refresh') }}</span>
        </button>
      </div>
    </div>

    <div v-if="loading" class="flex min-h-48 items-center justify-center text-sm text-gray-500 dark:text-dark-400">
      {{ t('usage.performance.loading') }}
    </div>
    <div v-else-if="error" class="border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-300">
      {{ error }}
    </div>
    <template v-else-if="report">
      <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500 dark:text-dark-400">
        <span>{{ t('usage.performance.readOnly') }}</span>
        <span>{{ t('usage.performance.warningLegend') }}</span>
      </div>
      <PerformanceTable :title="t('usage.performance.groups')" :rows="report.groups" :thresholds="report.thresholds" />
      <PerformanceTable :title="t('usage.performance.accounts')" :rows="report.accounts" :thresholds="report.thresholds" />
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PerformanceTable from './UpstreamPerformanceTable.vue'
import type { UpstreamPerformanceReport } from '@/api/admin/usage'

const props = defineProps<{
  report: UpstreamPerformanceReport | null
  loading: boolean
  error: string
  period: '24h' | '7d'
}>()
const emit = defineEmits<{
  refresh: []
  'update:period': [value: '24h' | '7d']
}>()
const { t } = useI18n()
const period = computed(() => props.period)
const periods = computed(() => [
  { value: '24h' as const, label: t('usage.performance.last24h') },
  { value: '7d' as const, label: t('usage.performance.last7d') },
])
const selectPeriod = (value: '24h' | '7d') => {
  if (value === props.period) return
  emit('update:period', value)
  emit('refresh')
}
</script>
