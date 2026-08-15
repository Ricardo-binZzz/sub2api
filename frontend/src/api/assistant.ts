import { apiClient } from './client'

export interface AssistantStatus {
  enabled: boolean
  model?: string
}

export interface AssistantReply {
  answer: string
}

function basePath(isAdmin: boolean): string {
  return isAdmin ? '/admin/assistant' : '/assistant'
}

export async function getAssistantStatus(isAdmin: boolean): Promise<AssistantStatus> {
  const { data } = await apiClient.get<AssistantStatus>(`${basePath(isAdmin)}/status`)
  return data
}

export async function chatWithAssistant(isAdmin: boolean, question: string): Promise<AssistantReply> {
  const { data } = await apiClient.post<AssistantReply>(`${basePath(isAdmin)}/chat`, { question })
  return data
}
