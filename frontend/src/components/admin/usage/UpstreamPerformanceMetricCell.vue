<template>
  <td
    class="whitespace-nowrap px-4 py-3"
    :class="isWarning ? 'font-semibold text-amber-600 dark:text-amber-400' : ''"
  >
    {{ value == null ? '-' : `${Math.round(value)} ms` }}
  </td>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value?: number | null; warning?: number }>()
const isWarning = computed(
  () => props.value != null && props.warning != null && props.value >= props.warning
)
</script>
