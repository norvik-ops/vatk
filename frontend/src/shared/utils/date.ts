/**
 * Date formatting utilities with user timezone support.
 */

export function formatDateTime(dateStr: string): string {
  return new Intl.DateTimeFormat('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZoneName: 'short',
  }).format(new Date(dateStr))
}

export function formatDate(dateStr: string): string {
  return new Intl.DateTimeFormat('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  }).format(new Date(dateStr))
}

export function formatRelative(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 60) return `vor ${String(minutes)} Minuten`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `vor ${String(hours)} Stunden`
  const days = Math.floor(hours / 24)
  if (days < 7) return `vor ${String(days)} Tagen`
  return formatDate(dateStr)
}
