<template>
  <AppLayout>
    <div class="mx-auto flex min-h-[calc(100vh-9rem)] max-w-5xl flex-col">
      <header class="mb-5 flex flex-col gap-3 border-b border-gray-200 pb-5 dark:border-dark-700 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2">
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('assistant.title') }}</h1>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400">
            {{ isAdmin ? t('assistant.subtitleAdmin') : t('assistant.subtitleUser') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span class="rounded border border-gray-200 px-2 py-1 dark:border-dark-600">
            {{ isAdmin ? t('assistant.operationsContext') : t('assistant.personalContext') }}
          </span>
          <span v-if="status?.model">{{ t('assistant.model', { model: status.model }) }}</span>
          <button
            v-if="messages.length > 0"
            type="button"
            class="inline-flex h-8 w-8 items-center justify-center rounded border border-gray-200 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:hover:bg-dark-700 dark:hover:text-white"
            :title="t('assistant.clearConversation')"
            :aria-label="t('assistant.clearConversation')"
            :disabled="sending"
            @click="clearConversation"
          >
            <Icon name="trash" size="sm" />
          </button>
        </div>
      </header>

      <div v-if="statusLoading" class="flex flex-1 items-center justify-center text-sm text-gray-500">
        <span class="h-5 w-5 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600" aria-hidden="true"></span>
      </div>

      <div v-else-if="!status?.enabled" class="flex flex-1 items-center justify-center py-12">
        <div class="max-w-md text-center">
          <Icon name="chat" size="xl" class="mx-auto mb-4 text-gray-400" />
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('assistant.unavailableTitle') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
            {{ statusError ? t('assistant.loadError') : (isAdmin ? t('assistant.unavailableAdmin') : t('assistant.unavailableUser')) }}
          </p>
          <button type="button" class="btn btn-secondary mt-5" @click="loadStatus">
            <Icon name="refresh" size="sm" />
            {{ t('assistant.retry') }}
          </button>
        </div>
      </div>

      <template v-else>
        <section ref="messagePane" class="min-h-0 flex-1 overflow-y-auto py-2" aria-live="polite">
          <div v-if="messages.length === 0" class="flex min-h-[18rem] items-center justify-center">
            <div class="max-w-xl text-center">
              <Icon name="sparkles" size="xl" class="mx-auto mb-4 text-primary-500" />
              <p class="text-base leading-7 text-gray-700 dark:text-gray-300">
                {{ isAdmin ? t('assistant.welcomeAdmin') : t('assistant.welcomeUser') }}
              </p>
            </div>
          </div>

          <div v-else class="space-y-5">
            <div v-for="message in messages" :key="message.id" class="flex" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
              <div
                class="max-w-[88%] break-words px-4 py-3 text-sm leading-6 sm:max-w-[75%]"
                :class="message.role === 'user'
                  ? 'whitespace-pre-wrap rounded-lg bg-primary-600 text-white'
                  : 'border-l-2 border-primary-500 bg-gray-50 text-gray-800 dark:bg-dark-800 dark:text-gray-200'"
              >
                <div
                  v-if="message.role === 'assistant'"
                  class="assistant-markdown"
                  v-html="renderMarkdown(message.content)"
                ></div>
                <template v-else>{{ message.content }}</template>
              </div>
            </div>
            <div v-if="sending" class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <span class="h-4 w-4 animate-spin rounded-full border-2 border-gray-300 border-t-primary-600" aria-hidden="true"></span>
              {{ t('assistant.sending') }}
            </div>
          </div>
        </section>

        <div v-if="requestError" class="mb-3 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
          {{ t('assistant.requestError') }}
        </div>

        <footer class="border-t border-gray-200 pt-4 dark:border-dark-700">
          <div v-if="messages.length === 0" class="mb-3 flex flex-wrap gap-2">
            <button
              v-if="rewardStatus?.can_apply"
              type="button"
              class="inline-flex items-center gap-1.5 rounded border border-emerald-200 bg-emerald-50 px-3 py-1.5 text-left text-xs font-medium text-emerald-700 transition-colors hover:border-emerald-300 hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-300 dark:hover:bg-emerald-900/30"
              :disabled="sending"
              @click="claimReward"
            >
              <Icon name="gift" size="sm" />
              {{ t('assistant.reward.action') }}
            </button>
            <button
              v-for="prompt in quickPrompts"
              :key="prompt"
              type="button"
              class="rounded border border-gray-200 bg-white px-3 py-1.5 text-left text-xs font-medium text-gray-600 transition-colors hover:border-primary-300 hover:bg-primary-50 hover:text-primary-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-700 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
              :disabled="sending"
              @click="sendQuickPrompt(prompt)"
            >
              {{ prompt }}
            </button>
          </div>
          <form class="rounded-lg border border-gray-300 bg-white focus-within:border-primary-500 focus-within:ring-1 focus-within:ring-primary-500 dark:border-dark-600 dark:bg-dark-800" @submit.prevent="sendMessage">
            <textarea
              v-model="question"
              :placeholder="t('assistant.placeholder')"
              :maxlength="MAX_QUESTION_CHARS"
              rows="3"
              class="block max-h-40 min-h-20 w-full resize-y border-0 bg-transparent px-4 py-3 text-sm text-gray-900 outline-none placeholder:text-gray-400 focus:ring-0 dark:text-white"
              :disabled="sending"
              @keydown.enter="handleEnter"
            ></textarea>
            <div class="flex items-center justify-between border-t border-gray-100 px-3 py-2 dark:border-dark-700">
              <span class="text-xs text-gray-400">{{ t('assistant.remaining', { count: remainingChars }) }}</span>
              <button type="submit" class="btn btn-primary h-9" :disabled="!canSend">
                <Icon name="arrowUp" size="sm" />
                {{ t('assistant.send') }}
              </button>
            </div>
          </form>
        </footer>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Icon } from '@/components/icons'
import {
  applyAssistantReward,
  chatWithAssistant,
  getAssistantRewardStatus,
  getAssistantStatus,
  type AssistantHistoryMessage,
  type AssistantRewardStatus,
  type AssistantStatus
} from '@/api/assistant'
import { useAuthStore } from '@/stores/auth'
import { formatCurrency } from '@/utils/format'

const MAX_QUESTION_CHARS = 2000

interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
}

const { t } = useI18n()
const authStore = useAuthStore()
const isAdmin = computed(() => authStore.isAdmin)
const status = ref<AssistantStatus | null>(null)
const rewardStatus = ref<AssistantRewardStatus | null>(null)
const statusLoading = ref(true)
const statusError = ref(false)
const sending = ref(false)
const requestError = ref(false)
const question = ref('')
const messages = ref<ChatMessage[]>([])
const messagePane = ref<HTMLElement | null>(null)
let nextMessageID = 1

const remainingChars = computed(() => MAX_QUESTION_CHARS - Array.from(question.value).length)
const canSend = computed(() => !sending.value && question.value.trim().length > 0 && remainingChars.value >= 0)
const quickPrompts = computed(() => isAdmin.value
  ? [
      t('assistant.quickPrompts.adminTraffic'),
      t('assistant.quickPrompts.adminGroups'),
      t('assistant.quickPrompts.adminRisk'),
      t('assistant.quickPrompts.adminCost')
    ]
  : [
      t('assistant.quickPrompts.userCost'),
      t('assistant.quickPrompts.userErrors'),
      t('assistant.quickPrompts.userOptimize')
    ])

async function loadStatus() {
  statusLoading.value = true
  statusError.value = false
  try {
    status.value = await getAssistantStatus(isAdmin.value)
    if (!isAdmin.value && status.value.reward_enabled) {
      try {
        rewardStatus.value = await getAssistantRewardStatus()
      } catch {
        rewardStatus.value = null
      }
    } else {
      rewardStatus.value = null
    }
  } catch {
    status.value = null
    statusError.value = true
  } finally {
    statusLoading.value = false
  }
}

async function claimReward() {
  if (sending.value || !rewardStatus.value?.can_apply) return

  messages.value.push({ id: nextMessageID++, role: 'user', content: t('assistant.reward.request') })
  requestError.value = false
  sending.value = true
  await scrollToLatest()

  try {
    const result = await applyAssistantReward()
    rewardStatus.value = {
      enabled: true,
      can_apply: false,
      claimed: true,
      decision: result.decision
    }

    let content: string
    if (result.granted) {
      content = t('assistant.reward.granted', { amount: formatCurrency(result.decision.final_amount) })
      await authStore.refreshUser()
    } else if (result.decision.status === 'granted') {
      content = t('assistant.reward.alreadyGranted', { amount: formatCurrency(result.decision.final_amount) })
    } else {
      content = t('assistant.reward.notGranted')
    }
    messages.value.push({ id: nextMessageID++, role: 'assistant', content })
  } catch {
    requestError.value = true
  } finally {
    sending.value = false
    await scrollToLatest()
  }
}

async function scrollToLatest() {
  await nextTick()
  if (messagePane.value) messagePane.value.scrollTop = messagePane.value.scrollHeight
}

async function sendMessage() {
  if (!canSend.value) return
  const content = question.value.trim()
  const history: AssistantHistoryMessage[] = messages.value.slice(-10).map(({ role, content: historyContent }) => ({
    role,
    content: historyContent
  }))
  messages.value.push({ id: nextMessageID++, role: 'user', content })
  question.value = ''
  requestError.value = false
  sending.value = true
  await scrollToLatest()
  try {
    const reply = await chatWithAssistant(isAdmin.value, content, history)
    messages.value.push({ id: nextMessageID++, role: 'assistant', content: reply.answer })
  } catch {
    requestError.value = true
  } finally {
    sending.value = false
    await scrollToLatest()
  }
}

function sendQuickPrompt(prompt: string) {
  question.value = prompt
  void sendMessage()
}

function clearConversation() {
  messages.value = []
  requestError.value = false
  question.value = ''
}

function renderMarkdown(content: string): string {
  return DOMPurify.sanitize(marked.parse(content) as string)
}

function handleEnter(event: KeyboardEvent) {
  if (event.shiftKey || event.isComposing) return
  event.preventDefault()
  void sendMessage()
}

onMounted(loadStatus)
</script>

<style scoped>
.assistant-markdown :deep(p) {
  margin: 0 0 0.75rem;
}

.assistant-markdown :deep(p:last-child),
.assistant-markdown :deep(ul:last-child),
.assistant-markdown :deep(ol:last-child),
.assistant-markdown :deep(pre:last-child) {
  margin-bottom: 0;
}

.assistant-markdown :deep(ul),
.assistant-markdown :deep(ol) {
  margin: 0 0 0.75rem 1.25rem;
}

.assistant-markdown :deep(ul) {
  list-style: disc;
}

.assistant-markdown :deep(ol) {
  list-style: decimal;
}

.assistant-markdown :deep(li) {
  margin-top: 0.35rem;
  padding-left: 0.15rem;
}

.assistant-markdown :deep(code) {
  border-radius: 0.25rem;
  background: rgb(229 231 235 / 0.8);
  padding: 0.1rem 0.3rem;
  font-size: 0.8125rem;
}

.assistant-markdown :deep(pre) {
  margin: 0 0 0.75rem;
  overflow-x: auto;
  border-radius: 0.375rem;
  background: rgb(17 24 39);
  padding: 0.75rem;
  color: rgb(243 244 246);
}

.assistant-markdown :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}

:global(.dark) .assistant-markdown :deep(code) {
  background: rgb(55 65 81 / 0.85);
}
</style>
