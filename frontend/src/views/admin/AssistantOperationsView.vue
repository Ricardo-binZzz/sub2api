<template>
  <AppLayout>
    <main class="mx-auto w-full max-w-7xl space-y-6 p-4 sm:p-6">
      <header
        class="flex flex-col gap-3 border-b border-gray-200 pb-5 dark:border-gray-700 sm:flex-row sm:items-center sm:justify-between"
      >
        <div>
          <h1 class="text-xl font-semibold text-gray-950 dark:text-white">
            {{ t('assistant.operations.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500">{{ t('assistant.operations.subtitle') }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn-secondary" :disabled="busy" @click="planNow">
            {{ t('assistant.operations.planNow') }}
          </button>
          <button class="btn-primary" :disabled="busy" @click="reload">
            {{ t('common.refresh') }}
          </button>
        </div>
      </header>

      <p
        v-if="notice"
        role="status"
        class="border-l-2 px-3 py-2 text-sm"
        :class="notice.type === 'error' ? 'border-red-500 text-red-700 dark:text-red-400' : 'border-emerald-600 text-emerald-700 dark:text-emerald-400'"
      >
        {{ notice.message }}
      </p>

      <section
        class="grid grid-cols-2 gap-px overflow-hidden rounded-md border border-gray-200 bg-gray-200 dark:border-gray-700 dark:bg-gray-700 lg:grid-cols-4"
      >
        <div v-for="metric in metrics" :key="metric.label" class="bg-white p-4 dark:bg-gray-900">
          <div class="text-xs text-gray-500">{{ metric.label }}</div>
          <div class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">{{ metric.value }}</div>
        </div>
      </section>

      <section class="space-y-4 border-b border-gray-200 pb-6 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h2 class="section-title">{{ t('assistant.operations.policy') }}</h2>
          <button class="btn-primary" :disabled="busy" @click="savePolicy">
            {{ t('common.save') }}
          </button>
        </div>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <label
            v-for="field in toggles"
            :key="field.key"
            class="flex min-h-14 items-center justify-between gap-3 border-b border-gray-100 py-2 dark:border-gray-800"
          >
            <span class="text-sm text-gray-700 dark:text-gray-200">{{ field.label }}</span>
            <input v-model="policy[field.key]" type="checkbox" class="h-4 w-4 accent-emerald-600" />
          </label>
          <label v-for="field in numbers" :key="field.key" class="space-y-1">
            <span class="text-xs text-gray-500">{{ field.label }}</span>
            <input v-model.number="policy[field.key]" type="number" class="input" />
          </label>
        </div>
      </section>

      <section class="space-y-4 border-b border-gray-200 pb-6 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h2 class="section-title">{{ t('assistant.operations.connections') }}</h2>
          <button class="btn-secondary" :disabled="busy" @click="openConnection()">
            {{ t('assistant.operations.addConnection') }}
          </button>
        </div>
        <div
          v-if="connections.length"
          class="divide-y divide-gray-200 border-y border-gray-200 dark:divide-gray-700 dark:border-gray-700"
        >
          <div
            v-for="item in connections"
            :key="item.id"
            class="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-gray-900 dark:text-white">{{ item.name }}</span>
                <span class="badge">{{ item.platform }}</span>
                <span :class="item.enabled ? 'text-emerald-600' : 'text-gray-400'" class="text-xs">
                  {{ item.enabled ? t('common.enabled') : t('common.disabled') }}
                </span>
              </div>
              <p v-if="item.last_error" class="mt-1 max-w-2xl truncate text-xs text-red-600">
                {{ item.last_error }}
              </p>
            </div>
            <div class="flex gap-2">
              <button class="btn-plain" :disabled="busy" @click="testConnection(item.id)">
                {{ t('assistant.operations.test') }}
              </button>
              <button class="btn-plain" :disabled="busy" @click="openConnection(item)">
                {{ t('common.edit') }}
              </button>
              <button class="btn-danger" :disabled="busy" @click="removeConnection(item.id)">
                {{ t('common.delete') }}
              </button>
            </div>
          </div>
        </div>
        <p v-else class="empty">{{ t('assistant.operations.noConnections') }}</p>
      </section>

      <section class="grid gap-6 xl:grid-cols-[minmax(16rem,1fr)_minmax(0,2fr)]">
        <form class="space-y-3" @submit.prevent="createTask">
          <h2 class="section-title">{{ t('assistant.operations.manualTask') }}</h2>
          <select v-model.number="task.connection_id" class="input" required>
            <option :value="0" disabled>{{ t('assistant.operations.chooseConnection') }}</option>
            <option v-for="item in connections" :key="item.id" :value="item.id">{{ item.name }}</option>
          </select>
          <textarea v-model="task.content" rows="6" maxlength="3000" class="input resize-y" required />
          <button class="btn-primary" :disabled="busy">{{ t('assistant.operations.createTask') }}</button>
        </form>

        <div class="min-w-0">
          <h2 class="section-title mb-3">{{ t('assistant.operations.tasks') }}</h2>
          <div class="overflow-x-auto">
            <table class="w-full text-left text-sm">
              <thead class="border-b border-gray-200 text-xs text-gray-500 dark:border-gray-700">
                <tr>
                  <th class="py-2">ID</th>
                  <th>{{ t('assistant.operations.platform') }}</th>
                  <th>{{ t('assistant.operations.status') }}</th>
                  <th>{{ t('assistant.operations.content') }}</th>
                  <th class="text-right">{{ t('assistant.operations.actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                <tr v-for="item in tasks" :key="item.id">
                  <td class="py-3">{{ item.id }}</td>
                  <td>{{ item.platform }}</td>
                  <td><span class="badge">{{ item.status }}</span></td>
                  <td class="max-w-xs truncate pr-3" :title="item.content">{{ item.content }}</td>
                  <td class="whitespace-nowrap text-right">
                    <button
                      v-if="item.status === 'pending_approval'"
                      class="btn-plain"
                      :disabled="busy"
                      @click="taskAction(item.id, 'approve')"
                    >
                      {{ t('assistant.operations.approve') }}
                    </button>
                    <button
                      v-if="['approved', 'failed'].includes(item.status)"
                      class="btn-plain"
                      :disabled="busy"
                      @click="taskAction(item.id, 'run')"
                    >
                      {{ t('assistant.operations.run') }}
                    </button>
                    <button
                      v-if="!['succeeded', 'cancelled'].includes(item.status)"
                      class="btn-plain text-red-600"
                      :disabled="busy"
                      @click="taskAction(item.id, 'cancel')"
                    >
                      {{ t('common.cancel') }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section class="min-w-0 border-t border-gray-200 pt-5 dark:border-gray-700">
        <h2 class="section-title mb-3">{{ t('assistant.operations.runs') }}</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead class="border-b border-gray-200 text-xs text-gray-500 dark:border-gray-700">
              <tr>
                <th class="py-2">ID</th>
                <th>{{ t('assistant.operations.task') }}</th>
                <th>{{ t('assistant.operations.status') }}</th>
                <th>{{ t('assistant.operations.result') }}</th>
                <th>{{ t('assistant.operations.startedAt') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr v-for="item in runs" :key="item.id">
                <td class="py-3">{{ item.id }}</td>
                <td>#{{ item.task_id }}</td>
                <td><span class="badge">{{ item.status }}</span></td>
                <td class="max-w-xl truncate pr-3" :title="item.error || item.output">
                  {{ item.error || item.output || '-' }}
                </td>
                <td class="whitespace-nowrap">{{ formatTime(item.started_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="!runs.length" class="empty">{{ t('assistant.operations.noRuns') }}</p>
      </section>
    </main>

    <div
      v-if="showConnection"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      @click.self="showConnection = false"
    >
      <form
        class="w-full max-w-lg space-y-4 rounded-md bg-white p-5 shadow-xl dark:bg-gray-900"
        @submit.prevent="saveConnection"
      >
        <div class="flex items-center justify-between">
          <h2 class="section-title">{{ t('assistant.operations.connection') }}</h2>
          <button type="button" class="text-gray-500" aria-label="Close" @click="showConnection = false">&#10005;</button>
        </div>
        <input v-model="connection.name" class="input" :placeholder="t('assistant.operations.name')" required />
        <select v-model="connection.platform" class="input" :disabled="!!connection.id">
          <option value="bluesky">Bluesky</option>
          <option value="telegram">Telegram</option>
          <option value="discord">Discord</option>
        </select>
        <template v-if="connection.platform === 'bluesky'">
          <input v-model="connection.config.identifier" class="input" placeholder="handle.example.com" />
          <input v-model="connection.config.app_password" type="password" class="input" placeholder="App password" />
        </template>
        <template v-else-if="connection.platform === 'telegram'">
          <input v-model="connection.config.bot_token" type="password" class="input" placeholder="Bot token" />
          <input v-model="connection.config.chat_id" class="input" placeholder="Broadcast chat ID" />
          <input
            v-model="connection.config.admin_chat_ids"
            class="input"
            placeholder="Admin chat IDs, comma separated"
          />
        </template>
        <input
          v-else
          v-model="connection.config.webhook_url"
          type="password"
          class="input"
          placeholder="Discord webhook URL"
        />
        <label class="flex items-center gap-2 text-sm">
          <input v-model="connection.enabled" type="checkbox" />{{ t('common.enabled') }}
        </label>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn-secondary" @click="showConnection = false">{{ t('common.cancel') }}</button>
          <button class="btn-primary" :disabled="busy">{{ t('common.save') }}</button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  operationsAPI,
  type OperationConnection,
  type OperationPlatform,
  type OperationPolicy,
  type OperationRun,
  type OperationTask
} from '@/api/admin/operations'

const { t } = useI18n()
const busy = ref(false)
const notice = ref<{ type: 'success' | 'error'; message: string } | null>(null)
const summary = ref({ connections: 0, pending_tasks: 0, succeeded_today: 0 })
const connections = ref<OperationConnection[]>([])
const tasks = ref<OperationTask[]>([])
const runs = ref<OperationRun[]>([])
const policy = reactive<OperationPolicy>({
  enabled: false,
  autonomous_enabled: false,
  auto_publish: false,
  require_approval: true,
  planning_interval_minutes: 360,
  min_publish_interval_minutes: 180,
  max_daily_actions: 3,
  quiet_hours_start: 23,
  quiet_hours_end: 8
})
const task = reactive({ connection_id: 0, content: '' })
const showConnection = ref(false)
const connection = reactive<{
  id?: number
  platform: OperationPlatform
  name: string
  enabled: boolean
  config: Record<string, string>
}>({ platform: 'bluesky', name: '', enabled: false, config: {} })

const metrics = computed(() => [
  { label: t('assistant.operations.connections'), value: summary.value.connections },
  { label: t('assistant.operations.pending'), value: summary.value.pending_tasks },
  { label: t('assistant.operations.today'), value: summary.value.succeeded_today },
  {
    label: t('assistant.operations.mode'),
    value: policy.autonomous_enabled ? t('assistant.operations.autonomous') : t('assistant.operations.manual')
  }
])
const toggles = [
  { key: 'enabled' as const, label: t('assistant.operations.enableRuntime') },
  { key: 'autonomous_enabled' as const, label: t('assistant.operations.enableAutonomous') },
  { key: 'auto_publish' as const, label: t('assistant.operations.autoPublish') },
  { key: 'require_approval' as const, label: t('assistant.operations.requireApproval') }
]
const numbers = [
  { key: 'planning_interval_minutes' as const, label: t('assistant.operations.planningInterval') },
  { key: 'min_publish_interval_minutes' as const, label: t('assistant.operations.publishInterval') },
  { key: 'max_daily_actions' as const, label: t('assistant.operations.dailyBudget') },
  { key: 'quiet_hours_start' as const, label: t('assistant.operations.quietStart') },
  { key: 'quiet_hours_end' as const, label: t('assistant.operations.quietEnd') }
]

function errorMessage(error: unknown) {
  if (typeof error === 'object' && error && 'message' in error) return String(error.message)
  return t('assistant.operations.actionFailed')
}

async function perform(action: () => Promise<unknown>, success?: string) {
  busy.value = true
  notice.value = null
  try {
    await action()
    if (success) notice.value = { type: 'success', message: success }
  } catch (error) {
    notice.value = { type: 'error', message: errorMessage(error) }
  } finally {
    busy.value = false
  }
}

async function loadData() {
  const [nextSummary, nextConnections, nextPolicy, taskPage, runPage] = await Promise.all([
    operationsAPI.summary(),
    operationsAPI.connections(),
    operationsAPI.policy(),
    operationsAPI.tasks(),
    operationsAPI.runs()
  ])
  summary.value = nextSummary
  connections.value = nextConnections
  Object.assign(policy, nextPolicy)
  tasks.value = taskPage.items ?? []
  runs.value = runPage.items ?? []
}

async function reload() {
  await perform(loadData)
}

async function savePolicy() {
  await perform(async () => {
    Object.assign(policy, await operationsAPI.updatePolicy({ ...policy }))
    await loadData()
  }, t('assistant.operations.saved'))
}

function openConnection(item?: OperationConnection) {
  Object.assign(
    connection,
    item
      ? { id: item.id, platform: item.platform, name: item.name, enabled: item.enabled, config: {} }
      : { id: undefined, platform: 'bluesky', name: '', enabled: false, config: {} }
  )
  showConnection.value = true
}

async function saveConnection() {
  await perform(async () => {
    await operationsAPI.saveConnection(
      {
        platform: connection.platform,
        name: connection.name,
        enabled: connection.enabled,
        config: connection.config
      },
      connection.id
    )
    showConnection.value = false
    await loadData()
  }, t('assistant.operations.saved'))
}

async function removeConnection(id: number) {
  if (!confirm(t('assistant.operations.confirmDelete'))) return
  await perform(async () => {
    await operationsAPI.deleteConnection(id)
    await loadData()
  }, t('assistant.operations.deleted'))
}

async function testConnection(id: number) {
  await perform(async () => {
    await operationsAPI.testConnection(id)
    await loadData()
  }, t('assistant.operations.testSucceeded'))
}

async function createTask() {
  await perform(async () => {
    await operationsAPI.createTask({ ...task, kind: 'promotion' })
    task.content = ''
    await loadData()
  }, t('assistant.operations.taskCreated'))
}

async function taskAction(id: number, action: 'approve' | 'run' | 'cancel') {
  await perform(async () => {
    if (action === 'approve') await operationsAPI.approveTask(id)
    if (action === 'run') await operationsAPI.runTask(id)
    if (action === 'cancel') await operationsAPI.cancelTask(id)
    await loadData()
  }, t('assistant.operations.taskUpdated'))
}

async function planNow() {
  await perform(async () => {
    await operationsAPI.planNow()
    await loadData()
  }, t('assistant.operations.planCreated'))
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

onMounted(reload)
</script>

<style scoped>
.section-title { @apply text-sm font-semibold text-gray-950 dark:text-white; }
.input { @apply w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-emerald-600 focus:ring-1 focus:ring-emerald-600 dark:border-gray-700 dark:bg-gray-950 dark:text-white; }
.btn-primary { @apply inline-flex min-h-9 items-center justify-center rounded-md bg-emerald-700 px-3 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50; }
.btn-secondary { @apply inline-flex min-h-9 items-center justify-center rounded-md border border-gray-300 px-3 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800; }
.btn-plain { @apply px-2 py-1 text-sm text-gray-600 hover:text-gray-950 disabled:opacity-50 dark:text-gray-300 dark:hover:text-white; }
.btn-danger { @apply px-2 py-1 text-sm text-red-600 disabled:opacity-50; }
.badge { @apply inline-flex rounded bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-300; }
.empty { @apply border-y border-gray-200 py-8 text-center text-sm text-gray-500 dark:border-gray-700; }
</style>
