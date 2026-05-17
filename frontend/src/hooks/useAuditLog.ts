import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '../api/client'

export interface AuditLogEntry {
  id: string
  org_id: string
  user_id?: string
  user_email?: string
  action: string
  resource_type: string
  resource_id?: string
  resource_name?: string
  details?: Record<string, string>
  ip_address?: string
  created_at: string
}

export interface AuditLogResult {
  entries: AuditLogEntry[]
  total: number
}

export interface AuditLogFilters {
  /** Max entries to return (default 25, max 500) */
  limit?: number
  /** Entries to skip — drives server-side pagination */
  offset?: number
  /** RFC3339 timestamp — filter created_at >= from */
  from?: string
  /** RFC3339 timestamp — filter created_at <= to */
  to?: string
  /** Substring match on user_email (case-insensitive) */
  userEmail?: string
  /** Exact match on action field */
  action?: string
}

export function useAuditLog(filters: AuditLogFilters = {}) {
  const params = new URLSearchParams()

  if (filters.limit)     params.set('limit',      String(filters.limit))
  if (filters.offset)    params.set('offset',     String(filters.offset))
  if (filters.from)      params.set('from',       filters.from)
  if (filters.to)        params.set('to',         filters.to)
  if (filters.userEmail) params.set('user_email', filters.userEmail)
  if (filters.action)    params.set('action',     filters.action)

  return useQuery<AuditLogResult>({
    queryKey: ['audit-log', filters],
    queryFn: () => {
      const qs = params.toString()
      return apiFetch<AuditLogResult>(`/audit-log${qs ? `?${qs}` : ''}`)
    },
    staleTime: 30_000,
  })
}
