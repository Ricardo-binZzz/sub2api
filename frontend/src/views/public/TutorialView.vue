<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="sticky top-0 z-40 border-b border-gray-200/80 bg-white/95 backdrop-blur dark:border-dark-700 dark:bg-dark-900/95">
      <div class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <img :src="siteLogo" alt="" class="h-9 w-9 rounded-lg object-contain" />
          <div class="min-w-0">
            <div class="truncate text-base font-semibold">{{ siteName }}</div>
            <div class="text-xs text-gray-500 dark:text-dark-400">使用教程</div>
          </div>
        </router-link>

        <div class="flex items-center gap-2">
          <a
            href="#quick-start"
            class="hidden rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800 sm:inline-flex"
          >
            快速开始
          </a>
          <router-link :to="primaryPath" class="btn btn-primary">
            <Icon :name="isLoggedIn ? 'grid' : 'login'" size="sm" />
            {{ isLoggedIn ? '进入控制台' : '登录' }}
          </router-link>
        </div>
      </div>
    </header>

    <div class="mx-auto grid max-w-7xl grid-cols-1 gap-8 px-4 py-8 sm:px-6 lg:grid-cols-[220px_minmax(0,1fr)] lg:py-12">
      <aside class="hidden lg:block">
        <nav class="sticky top-24 space-y-1 border-l border-gray-200 pl-4 text-sm dark:border-dark-700" aria-label="教程目录">
          <a
            v-for="item in toc"
            :key="item.href"
            :href="item.href"
            class="block rounded-r-md px-3 py-2 text-gray-600 transition-colors hover:bg-gray-100 hover:text-primary-700 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-primary-300"
          >
            {{ item.label }}
          </a>
        </nav>
      </aside>

      <main class="min-w-0 max-w-4xl">
        <section class="border-b border-gray-200 pb-10 dark:border-dark-700">
          <div class="mb-4 inline-flex items-center gap-2 rounded-md bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
            <Icon name="book" size="sm" />
            新用户上手
          </div>
          <h1 class="text-3xl font-bold tracking-normal sm:text-4xl">Bin API 使用教程</h1>
          <p class="mt-4 max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300">
            从注册、充值和创建 API 密钥开始，完成 Claude Code、Codex 或 OpenAI Responses API 的首次调用。
          </p>
          <div class="mt-6 flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-gray-500 dark:text-dark-400">
            <span class="inline-flex items-center gap-1.5"><Icon name="clock" size="sm" />约 5 分钟</span>
            <span class="inline-flex items-center gap-1.5"><Icon name="globe" size="sm" />API 地址：{{ apiBaseUrl }}</span>
          </div>
        </section>

        <section id="quick-start" class="scroll-mt-24 py-10">
          <SectionHeading eyebrow="01" title="使用流程" description="按顺序完成下面六步，首次调用成功后再配置额度和安全限制。" />
          <ol class="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <li v-for="(step, index) in quickSteps" :key="step.title" class="card flex gap-3 p-4">
              <span class="flex h-7 w-7 flex-none items-center justify-center rounded-md bg-primary-600 text-xs font-bold text-white">
                {{ index + 1 }}
              </span>
              <div>
                <div class="text-sm font-semibold">{{ step.title }}</div>
                <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ step.description }}</p>
              </div>
            </li>
          </ol>
        </section>

        <section id="account" class="scroll-mt-24 border-t border-gray-200 py-10 dark:border-dark-700">
          <SectionHeading eyebrow="02" title="注册与获取额度" description="余额或有效订阅决定密钥是否可以持续调用。" />
          <div class="mt-6 space-y-5">
            <GuideStep number="1" title="创建账户">
              打开注册页，填写邮箱和密码。注册后登录控制台；若站点启用了邮箱验证，请先完成验证。
              <template #action><router-link to="/register" class="text-link">前往注册</router-link></template>
            </GuideStep>
            <GuideStep number="2" title="充值、购买订阅或兑换">
              进入“充值/订阅”选择套餐并支付；已有兑换码时可直接在“兑换”页使用。完成后确认右上角余额或“我的订阅”中的额度已更新。
              <template #action>
                <router-link to="/purchase" class="text-link">充值/订阅</router-link>
                <router-link to="/redeem" class="text-link">使用兑换码</router-link>
              </template>
            </GuideStep>
            <GuideStep number="3" title="确认渠道状态">
              在“渠道状态”查看可用率和延迟。优先选择状态正常、首 Token 延迟较低的分组；模型和倍率以密钥页实际可选内容为准。
              <template #action><router-link to="/monitor" class="text-link">查看渠道状态</router-link></template>
            </GuideStep>
          </div>
        </section>

        <section id="api-key" class="scroll-mt-24 border-t border-gray-200 py-10 dark:border-dark-700">
          <SectionHeading eyebrow="03" title="创建 API 密钥" description="一个密钥对应一个调用入口，可单独设置分组、额度、限速和 IP 规则。" />
          <div class="mt-6 grid gap-6 md:grid-cols-[1fr_280px]">
            <div class="space-y-4">
              <GuideStep number="1" title="打开 API 密钥页">点击“创建密钥”，填写便于识别的名称，例如“个人电脑”或“生产服务”。</GuideStep>
              <GuideStep number="2" title="选择分组">分组决定可用模型、计费倍率和上游渠道。没有分组的密钥不能发起请求。</GuideStep>
              <GuideStep number="3" title="设置限制">个人使用建议设置总额度；生产环境再配置并发、时间窗口限额和 IP 白名单。</GuideStep>
              <GuideStep number="4" title="复制并妥善保存">完整密钥只应放在本机环境变量或密钥管理服务中，不要提交到 Git 仓库或发送到聊天记录。</GuideStep>
            </div>
            <div class="rounded-lg border border-primary-200 bg-primary-50 p-5 dark:border-primary-900 dark:bg-primary-950/30">
              <Icon name="key" size="lg" class="text-primary-600 dark:text-primary-300" />
              <div class="mt-4 text-sm font-semibold">最快的配置方式</div>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                创建后点击该密钥右侧的“使用密钥”。系统会根据分组平台生成当前地址、真实密钥和对应客户端配置。
              </p>
              <router-link to="/keys" class="btn btn-primary mt-5 w-full justify-center">打开 API 密钥页</router-link>
            </div>
          </div>
        </section>

        <section id="clients" class="scroll-mt-24 border-t border-gray-200 py-10 dark:border-dark-700">
          <SectionHeading eyebrow="04" title="接入客户端" description="下面使用占位符演示。实际接入时优先复制密钥页“使用密钥”弹窗生成的配置。" />

          <div class="mt-6 flex flex-col gap-4">
            <div class="inline-flex w-full overflow-x-auto rounded-lg bg-gray-100 p-1 dark:bg-dark-800 sm:w-fit" role="tablist">
              <button
                v-for="client in clientTabs"
                :key="client.id"
                type="button"
                class="whitespace-nowrap rounded-md px-4 py-2 text-sm font-medium transition-colors"
                :class="selectedClient === client.id ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-600 dark:text-dark-300'"
                @click="selectedClient = client.id"
              >
                {{ client.label }}
              </button>
            </div>

            <div v-if="selectedClient !== 'codex'" class="inline-flex w-fit rounded-lg border border-gray-200 p-1 dark:border-dark-700">
              <button
                v-for="os in osTabs"
                :key="os.id"
                type="button"
                class="rounded-md px-3 py-1.5 text-xs font-medium"
                :class="selectedOs === os.id ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900' : 'text-gray-500 dark:text-dark-300'"
                @click="selectedOs = os.id"
              >
                {{ os.label }}
              </button>
            </div>

            <div class="overflow-hidden rounded-lg border border-gray-800 bg-gray-950 text-gray-100">
              <div class="flex items-center justify-between border-b border-gray-800 bg-gray-900 px-4 py-2.5">
                <span class="truncate font-mono text-xs text-gray-400">{{ currentConfig.path }}</span>
                <button type="button" class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-gray-300 hover:bg-gray-800 hover:text-white" @click="copyCurrentConfig">
                  <Icon :name="copied ? 'check' : 'copy'" size="sm" />
                  {{ copied ? '已复制' : '复制' }}
                </button>
              </div>
              <pre class="overflow-x-auto p-4 text-sm leading-6"><code>{{ currentConfig.content }}</code></pre>
            </div>

            <div class="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
              <Icon name="exclamationCircle" size="md" class="mt-0.5 flex-none" />
              <p>把 <code class="font-mono">YOUR_API_KEY</code> 和 <code class="font-mono">MODEL_ID</code> 替换为密钥页显示的实际值。不要把示例占位符直接用于调用。</p>
            </div>
          </div>
        </section>

        <section id="verify" class="scroll-mt-24 border-t border-gray-200 py-10 dark:border-dark-700">
          <SectionHeading eyebrow="05" title="验证首次调用" description="成功标准是客户端收到模型输出，同时“使用记录”出现对应请求和扣费。" />
          <div class="mt-6 grid gap-4 sm:grid-cols-3">
            <VerificationItem icon="terminal" title="客户端有输出">确认没有 401、403、429 或模型不存在错误。</VerificationItem>
            <VerificationItem icon="chart" title="记录已生成">在使用记录中核对模型、Token、端点和实际消费。</VerificationItem>
            <VerificationItem icon="dollar" title="余额扣减正确">按分组倍率核对实际消费，避免选错高倍率渠道。</VerificationItem>
          </div>
          <div class="mt-5 flex flex-wrap gap-3">
            <router-link to="/usage" class="btn btn-primary">查看使用记录</router-link>
            <router-link to="/subscriptions" class="btn btn-secondary">查看订阅额度</router-link>
          </div>
        </section>

        <section id="troubleshooting" class="scroll-mt-24 border-t border-gray-200 py-10 dark:border-dark-700">
          <SectionHeading eyebrow="06" title="常见问题" description="先根据状态码定位，不要反复重试消耗额度。" />
          <dl class="mt-6 divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-700 dark:border-dark-700">
            <div v-for="item in troubleshooting" :key="item.code" class="grid gap-2 py-5 sm:grid-cols-[88px_1fr]">
              <dt><code class="rounded bg-gray-100 px-2 py-1 text-sm font-semibold dark:bg-dark-800">{{ item.code }}</code></dt>
              <dd>
                <div class="text-sm font-semibold">{{ item.title }}</div>
                <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ item.description }}</p>
              </dd>
            </div>
          </dl>
        </section>

        <section id="security" class="scroll-mt-24 border-t border-gray-200 py-10 dark:border-dark-700">
          <SectionHeading eyebrow="07" title="上线前检查" description="生产密钥应遵循最小权限，并且可以独立撤销。" />
          <ul class="mt-6 grid gap-3 sm:grid-cols-2">
            <li v-for="item in securityChecklist" :key="item" class="flex items-start gap-3 rounded-lg border border-gray-200 bg-white p-4 text-sm leading-6 dark:border-dark-700 dark:bg-dark-900">
              <Icon name="checkCircle" size="md" class="mt-0.5 flex-none text-emerald-600" />
              {{ item }}
            </li>
          </ul>
        </section>

        <footer class="border-t border-gray-200 py-8 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
          <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
            <span>教程地址与功能以当前 {{ siteName }} 控制台为准。</span>
            <span>需要协助：{{ contactInfo || '请联系站点管理员' }}</span>
          </div>
        </footer>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref } from 'vue'
import { useAppStore, useAuthStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

type ClientId = 'claude' | 'codex' | 'api'
type OsId = 'unix' | 'powershell'

const appStore = useAppStore()
const authStore = useAuthStore()
const selectedClient = ref<ClientId>('claude')
const selectedOs = ref<OsId>('unix')
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Bin API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '/logo.svg', { allowRelative: true, allowDataUrl: true }))
const contactInfo = computed(() => appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '')
const apiBaseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url || ''
  const fallback = typeof window === 'undefined' ? 'https://api.example.com' : window.location.origin
  return (configured || fallback).replace(/\/+$/, '')
})
const isLoggedIn = computed(() => Boolean(authStore.user))
const primaryPath = computed(() => isLoggedIn.value ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')

const toc = [
  { href: '#quick-start', label: '使用流程' },
  { href: '#account', label: '注册与额度' },
  { href: '#api-key', label: '创建 API 密钥' },
  { href: '#clients', label: '接入客户端' },
  { href: '#verify', label: '验证首次调用' },
  { href: '#troubleshooting', label: '常见问题' },
  { href: '#security', label: '上线前检查' },
]

const quickSteps = [
  { title: '注册并登录', description: '创建账户并进入用户控制台。' },
  { title: '获取额度', description: '充值、购买订阅或使用兑换码。' },
  { title: '查看渠道', description: '比较可用率、延迟和计费倍率。' },
  { title: '创建密钥', description: '选择分组并设置额度与安全限制。' },
  { title: '配置客户端', description: '复制站内生成的 Claude Code 或 Codex 配置。' },
  { title: '核对账单', description: '在使用记录中确认调用和扣费。' },
]

const clientTabs: Array<{ id: ClientId; label: string }> = [
  { id: 'claude', label: 'Claude Code' },
  { id: 'codex', label: 'Codex' },
  { id: 'api', label: 'OpenAI API' },
]
const osTabs: Array<{ id: OsId; label: string }> = [
  { id: 'unix', label: 'macOS / Linux' },
  { id: 'powershell', label: 'PowerShell' },
]

const currentConfig = computed(() => {
  const base = apiBaseUrl.value
  if (selectedClient.value === 'claude') {
    if (selectedOs.value === 'powershell') {
      return {
        path: 'PowerShell',
        content: `npm install -g @anthropic-ai/claude-code\n$env:ANTHROPIC_BASE_URL="${base}"\n$env:ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"\n$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC="1"\nclaude`,
      }
    }
    return {
      path: 'Terminal',
      content: `npm install -g @anthropic-ai/claude-code\nexport ANTHROPIC_BASE_URL="${base}"\nexport ANTHROPIC_AUTH_TOKEN="YOUR_API_KEY"\nexport CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC="1"\nclaude`,
    }
  }

  if (selectedClient.value === 'codex') {
    return {
      path: '~/.codex/config.toml + ~/.codex/auth.json',
      content: `# ~/.codex/config.toml\nmodel_provider = "binapi"\nmodel = "MODEL_ID"\n\n[model_providers.binapi]\nname = "Bin API"\nbase_url = "${base}"\nwire_api = "responses"\nrequires_openai_auth = true\n\n# ~/.codex/auth.json\n{\n  "OPENAI_API_KEY": "YOUR_API_KEY"\n}\n\n# Install and start\nnpm install -g @openai/codex\ncodex`,
    }
  }

  if (selectedOs.value === 'powershell') {
    return {
      path: 'PowerShell',
      content: `$headers = @{ Authorization = "Bearer YOUR_API_KEY" }\n$body = @{ model = "MODEL_ID"; input = "Reply with: Bin API is ready" } | ConvertTo-Json\nInvoke-RestMethod -Method Post -Uri "${base}/v1/responses" -Headers $headers -ContentType "application/json" -Body $body`,
    }
  }
  return {
    path: 'Terminal',
    content: `curl "${base}/v1/responses" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{"model":"MODEL_ID","input":"Reply with: Bin API is ready"}'`,
  }
})

const troubleshooting = [
  { code: '401', title: '密钥无效', description: '重新复制完整 API Key，检查环境变量是否仍是 YOUR_API_KEY，并确认密钥没有被禁用或删除。' },
  { code: '403', title: '分组或访问规则不允许', description: '确认密钥已选择分组；若配置了 IP 白名单，检查当前出口 IP；同时确认客户端类型符合渠道要求。' },
  { code: '429', title: '达到并发、速率或上游额度限制', description: '停止并发重试，等待窗口重置；在密钥页检查额度，在渠道状态页确认上游是否拥堵。' },
  { code: '404', title: '地址、端点或模型不匹配', description: 'Base URL 使用站点根地址；确认客户端没有重复拼接 /v1，并使用当前分组实际支持的 MODEL_ID。' },
  { code: '5xx', title: '上游或网关暂时异常', description: '先查看渠道状态并更换健康分组；保留请求时间和请求 ID，持续失败时联系管理员。' },
]

const securityChecklist = [
  '开发、测试和生产环境分别使用独立密钥。',
  '为每个密钥设置合理的总额度与时间窗口限额。',
  '服务端调用使用环境变量或密钥管理服务，不写入前端代码。',
  '生产服务配置固定出口 IP 后启用 IP 白名单。',
  '发现泄露后立即禁用旧密钥并创建新密钥。',
  '定期核对使用记录中的模型、IP、消费和失败请求。',
]

async function copyCurrentConfig() {
  await navigator.clipboard.writeText(currentConfig.value.content)
  copied.value = true
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => { copied.value = false }, 1600)
}

const SectionHeading = defineComponent({
  props: { eyebrow: { type: String, required: true }, title: { type: String, required: true }, description: { type: String, required: true } },
  setup(props) {
    return () => h('div', [
      h('div', { class: 'text-xs font-semibold text-primary-600 dark:text-primary-400' }, props.eyebrow),
      h('h2', { class: 'mt-1 text-2xl font-bold tracking-normal' }, props.title),
      h('p', { class: 'mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300' }, props.description),
    ])
  },
})

const GuideStep = defineComponent({
  props: { number: { type: String, required: true }, title: { type: String, required: true } },
  setup(props, { slots }) {
    return () => h('div', { class: 'flex gap-4' }, [
      h('span', { class: 'flex h-8 w-8 flex-none items-center justify-center rounded-md border border-gray-200 bg-white text-xs font-bold text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-primary-300' }, props.number),
      h('div', { class: 'min-w-0' }, [
        h('h3', { class: 'text-sm font-semibold' }, props.title),
        h('div', { class: 'mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300' }, slots.default?.()),
        slots.action ? h('div', { class: 'mt-2 flex flex-wrap gap-4 text-sm font-medium' }, slots.action()) : null,
      ]),
    ])
  },
})

const VerificationItem = defineComponent({
  props: { icon: { type: String, required: true }, title: { type: String, required: true } },
  setup(props, { slots }) {
    return () => h('div', { class: 'rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900' }, [
      h(Icon, { name: props.icon as 'terminal' | 'chart' | 'dollar', size: 'lg', class: 'text-primary-600 dark:text-primary-300' }),
      h('h3', { class: 'mt-3 text-sm font-semibold' }, props.title),
      h('div', { class: 'mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400' }, slots.default?.()),
    ])
  },
})
</script>

<style scoped>
.text-link {
  @apply inline-flex items-center text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200;
}
</style>
