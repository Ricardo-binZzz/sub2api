import { apiClient } from './client'

export interface AssistantStatus {
  enabled: boolean
  model?: string
  reward_enabled: boolean
}

export type AssistantRewardDecisionStatus = 'granted' | 'rejected'

export interface AssistantRewardDecision {
  id: number
  user_id: number
  campaign: string
  ai_suggested_amount: number
  ai_reason: string
  ai_confidence: number
  eligible: boolean
  rule_reasons: string[]
  final_amount: number
  status: AssistantRewardDecisionStatus
  balance_before?: number
  balance_after?: number
  created_at: string
  updated_at: string
}

export interface AssistantRewardStatus {
  enabled: boolean
  can_apply: boolean
  claimed: boolean
  reason?: string
  decision?: AssistantRewardDecision
}

export interface AssistantRewardApplyResponse {
  decision: AssistantRewardDecision
  granted: boolean
}

export interface AssistantReply {
  answer: string
}

export interface AssistantHistoryMessage {
  role: 'user' | 'assistant'
  content: string
}

function basePath(isAdmin: boolean): string {
  return isAdmin ? '/admin/assistant' : '/assistant'
}

export async function getAssistantStatus(isAdmin: boolean): Promise<AssistantStatus> {
  const { data } = await apiClient.get<AssistantStatus>(`${basePath(isAdmin)}/status`)
  return data
}

export async function chatWithAssistant(
  isAdmin: boolean,
  question: string,
  history: AssistantHistoryMessage[] = []
): Promise<AssistantReply> {
  const { data } = await apiClient.post<AssistantReply>(`${basePath(isAdmin)}/chat`, { question, history })
  return data
}

export async function getAssistantRewardStatus(): Promise<AssistantRewardStatus> {
  const { data } = await apiClient.get<AssistantRewardStatus>('/assistant/reward')
  return data
}

export async function applyAssistantReward(): Promise<AssistantRewardApplyResponse> {
  const { data } = await apiClient.post<AssistantRewardApplyResponse>('/assistant/reward/apply')
  return data
}
