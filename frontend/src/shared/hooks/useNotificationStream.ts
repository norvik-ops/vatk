import { useEffect, useRef, useState } from 'react'

/**
 * useNotificationStream — abonniert den Backend-SSE-Endpoint
 * `GET /api/v1/dashboard/notifications/stream` und liefert neue Notifications
 * an die Komponente via Callback. Sprint 17 / S17-3.
 *
 * Pattern: ADR-0019. Wir lehnen uns an `useAIStream` (Sprint 15) an, aber für
 * non-AI-Frames mit JSON-Payloads ohne `[DONE]`-Terminator (Stream lebt
 * solange die Session lebt).
 *
 * Lifecycle:
 *  - Auto-Connect beim Mount.
 *  - Auto-Reconnect mit 1-s-Backoff bei Disconnect (Netzwerk-Wechsel,
 *    Reverse-Proxy-Restart).
 *  - Cleanup beim Unmount.
 *
 * Heartbeat-Pongs (`event: ping`) werden ignoriert — sind nur fuer den
 * Proxy-Keepalive da, kein Daten-Event.
 */

export interface NotificationStreamItem {
  id: string
  title: string
  body: string
  type: string
  module: string
  read: boolean
  created_at: string
}

interface UseNotificationStreamOptions {
  /** Pfad relativ zu /api/v1 — Default ist die Dashboard-Notifications-Stream. */
  endpoint?: string
  /** Wird pro neuer Notification aufgerufen. */
  onItem: (item: NotificationStreamItem) => void
  /** Wenn false, baut der Hook keine Connection auf. Wechsel auf true reconnected. */
  enabled?: boolean
}

export function useNotificationStream({
  endpoint = '/dashboard/notifications/stream',
  onItem,
  enabled = true,
}: UseNotificationStreamOptions) {
  const [connected, setConnected] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  // onItem wird in useEffect-deps NICHT gelistet — wir nutzen ein Ref, damit
  // ein neuer Callback-Reference die Connection nicht abreißt.
  const onItemRef = useRef(onItem)
  onItemRef.current = onItem

  useEffect(() => {
    if (!enabled) return
    let cancelled = false

    const connect = async () => {
      while (!cancelled) {
        const controller = new AbortController()
        abortRef.current = controller
        try {
          const res = await fetch(`/api/v1${endpoint}`, {
            credentials: 'include',
            headers: { Accept: 'text/event-stream' },
            signal: controller.signal,
          })
          if (!res.ok || !res.body) {
            throw new Error(`stream HTTP ${res.status}`)
          }
          setConnected(true)
          const reader = res.body.getReader()
          const decoder = new TextDecoder()
          let buffer = ''
          while (!cancelled) {
            const { value, done } = await reader.read()
            if (done) break
            buffer += decoder.decode(value, { stream: true })
            let idx = buffer.indexOf('\n\n')
            while (idx !== -1) {
              const frame = buffer.slice(0, idx).trim()
              buffer = buffer.slice(idx + 2)
              // Heartbeat-Pongs ignorieren.
              if (frame.startsWith('event: ping')) {
                idx = buffer.indexOf('\n\n')
                continue
              }
              if (frame.startsWith('data: ')) {
                const payload = frame.slice(6)
                try {
                  const obj = JSON.parse(payload) as NotificationStreamItem
                  if (obj.id) onItemRef.current(obj)
                } catch {
                  // ungueltige Frame — uebersprungen
                }
              }
              idx = buffer.indexOf('\n\n')
            }
          }
        } catch (e: unknown) {
          if (e instanceof Error && e.name === 'AbortError') return
          // Netzwerk-Disconnect — Backoff 1 s, dann reconnect.
        }
        setConnected(false)
        if (!cancelled) await new Promise((r) => setTimeout(r, 1000))
      }
    }
    void connect()

    return () => {
      cancelled = true
      abortRef.current?.abort()
    }
  }, [endpoint, enabled])

  return { connected }
}
