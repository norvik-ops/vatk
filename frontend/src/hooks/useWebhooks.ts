import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../api/client'
import { toast } from '../shared/hooks/useToast'

// ─── Types ────────────────────────────────────────────────────────────────────

export type WebhookEvent =
  | 'finding.created'
  | 'finding.severity_changed'
  | 'incident.created'
  | 'incident.status_changed'
  | 'control.status_changed'

export interface Webhook {
  id: string
  name: string
  url: string
  secret?: string
  events: WebhookEvent[]
  active: boolean
  last_triggered_at: string | null
  created_at: string
  updated_at: string
}

export interface WebhookListResponse {
  data: Webhook[]
}

export interface CreateWebhookInput {
  name: string
  url: string
  secret?: string
  events: WebhookEvent[]
  active: boolean
}

export type UpdateWebhookInput = Partial<CreateWebhookInput>

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useWebhooks() {
  return useQuery<WebhookListResponse>({
    queryKey: ['webhooks'],
    queryFn: () => apiFetch<WebhookListResponse>('/webhooks'),
    retry: false,
    staleTime: 30_000,
  })
}

export function useCreateWebhook() {
  const qc = useQueryClient()
  return useMutation<Webhook, Error, CreateWebhookInput>({
    mutationFn: (input) =>
      apiFetch<Webhook>('/webhooks', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['webhooks'] })
    },
  })
}

export function useUpdateWebhook(id: string) {
  const qc = useQueryClient()
  return useMutation<Webhook, Error, UpdateWebhookInput>({
    mutationFn: (input) =>
      apiFetch<Webhook>(`/webhooks/${id}`, {
        method: 'PUT',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['webhooks'] })
    },
  })
}

export function useDeleteWebhook() {
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (id) => apiFetch<void>(`/webhooks/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['webhooks'] })
    },
  })
}

export function useTestWebhook() {
  return useMutation<void, Error, string>({
    mutationFn: (id) =>
      apiFetch<void>(`/webhooks/${id}/test`, { method: 'POST' }),
    onSuccess: () => {
      toast('Test-Ping gesendet', 'success')
    },
    onError: (err) => {
      toast(`Test fehlgeschlagen: ${err.message}`, 'error')
    },
  })
}
