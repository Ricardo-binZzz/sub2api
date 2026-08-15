<template>
  <AppLayout>
    <div class="mx-auto flex min-h-[calc(100vh-9rem)] max-w-5xl flex-col">
      <header class="mb-5 flex flex-col gap-3 border-b border-gray-200 pb-5 dark:border-dark-700 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2">
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('assistant.title') }}</h1>
            <span class="rounded bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              {{ t('assistant.readOnly') }}
            </span>
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
                class="max-w-[88%] whitespace-pre-wrap break-words px-4 py-3 text-sm leading-6 sm:max-w-[75%]"
                :class="message.role === 'user'
                  ? 'rounded-lg bg-primary-600 text-white'
                  : 'border-l-2 border-primary-500 bg-gray-50 text-gray-800 dark:bg-dark-800 dark:text-gray-200'"
              >
                {{ message.content }}
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
          <div class="mt-2 flex flex-col gap-1 text-xs text-gray-500 dark:text-gray-400 sm:flex-row sm:justify-between">
            <span>{{ isAdmin ? t('assistant.privacyAdmin') : t('assistant.privacyUser') }}</span>
            <span>{{ t('assistant.disclaimer') }}</span>
          </div>
        </footer>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Icon } from '@/components/icons'
import { chatWithAssistant, getAssistantStatus, type AssistantStatus } from '@/api/assistant'
import { useAuthStore } from '@/stores/auth'

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

async function loadStatus() {
  statusLoading.value = true
  statusError.value = false
  try {
    status.value = await getAssistantStatus(isAdmin.value)
  } catch {
    status.value = null
    statusError.value = true
  } finally {
    statusLoading.value = false
  }
}

async function scrollToLatest() {
  await nextTick()
  if (messagePane.value) messagePane.value.scrollTop = messagePane.value.scrollHeight
}

async function sendMessage() {
  if (!canSend.value) return
  const content = question.value.trim()
  messages.value.push({ id: nextMessageID++, role: 'user', content })
  question.value = ''
  requestError.value = false
  sending.value = true
  await scrollToLatest()
  try {
    const reply = await chatWithAssistant(isAdmin.value, content)
    messages.value.push({ id: nextMessageID++, role: 'assistant', content: reply.answer })
  } catch {
    requestError.value = true
  } finally {
    sending.value = false
    await scrollToLatest()
  }
}

function handleEnter(event: KeyboardEvent) {
  if (event.shiftKey || event.isComposing) return
  event.preventDefault()
  void sendMessage()
}

onMounted(loadStatus)
</script>
