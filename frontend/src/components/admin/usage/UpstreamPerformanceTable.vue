<template>
  <div class="overflow-hidden border border-gray-200 dark:border-dark-700">
    <div class="border-b border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800/60">
      <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ title }}</h3>
    </div>
    <div class="overflow-x-auto">
      <table class="min-w-full text-left text-sm">
        <thead class="border-b border-gray-100 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
          <tr>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.name') }}</th>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.requests') }}</th>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.successRate') }}</th>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.firstP50') }}</th>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.firstP95') }}</th>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.totalP50') }}</th>
            <th class="whitespace-nowrap px-4 py-3">{{ t('usage.performance.totalP95') }}</th>
          </tr>
        </thead>
        <tbody v-if="rows.length" class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-for="row in rows" :key="row.id" class="text-gray-700 dark:text-dark-200">
            <td class="whitespace-nowrap px-4 py-3 font-medium">
              <span>{{ row.name }}</span>
              <span v-if="row.platform" class="ml-2 text-xs text-gray-400">{{ row.platform }}</span>
            </td>
            <td class="whitespace-nowrap px-4 py-3">{{ row.requests }}</td>
            <td
              class="whitespace-nowrap px-4 py-3"
              :class="row.success_rate < thresholds.success_rate_warning ? 'font-semibold text-red-600 dark:text-red-400' : ''"
            >
              {{ row.success_rate.toFixed(1) }}%
            </td>
            <MetricCell :value="row.first_token_p50_ms" :warning="thresholds.first_token_p50_warning_ms" />
            <MetricCell :value="row.first_token_p95_ms" :warning="thresholds.first_token_p95_warning_ms" />
            <MetricCell :value="row.duration_p50_ms" />
            <MetricCell :value="row.duration_p95_ms" />
          </tr>
        </tbody>
        <tbody v-else>
          <tr>
            <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('usage.performance.noData') }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UpstreamPerformanceMetric, UpstreamPerformanceReport } from '@/api/admin/usage'
import MetricCell from './UpstreamPerformanceMetricCell.vue'

defineProps<{
  title: string
  rows: UpstreamPerformanceMetric[]
  thresholds: UpstreamPerformanceReport['thresholds']
}>()
const { t } = useI18n()
</script>
