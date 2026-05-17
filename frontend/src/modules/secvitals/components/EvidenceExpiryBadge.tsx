import { Badge } from '../../../components/ui/badge'

interface EvidenceExpiryBadgeProps {
  expiresAt?: string | null
}

/**
 * EvidenceExpiryBadge renders an expiry indicator next to evidence items.
 *
 * - No badge:  no expiry date set.
 * - Yellow:    expires within the next 30 days — "Läuft ab: DD.MM.YYYY"
 * - Red:       already expired            — "Abgelaufen: DD.MM.YYYY"
 */
export function EvidenceExpiryBadge({ expiresAt }: EvidenceExpiryBadgeProps) {
  if (!expiresAt) return null

  const expiry = new Date(expiresAt)
  const now = new Date()
  const diffMs = expiry.getTime() - now.getTime()
  const diffDays = diffMs / (1000 * 60 * 60 * 24)

  const dateStr = expiry.toLocaleDateString('de-DE')

  if (diffMs < 0) {
    // Already expired
    return (
      <Badge variant="destructive" className="text-xs">
        Abgelaufen: {dateStr}
      </Badge>
    )
  }

  if (diffDays <= 30) {
    // Expires within 30 days — yellow warning
    return (
      <Badge variant="warning" className="text-xs">
        Läuft ab: {dateStr}
      </Badge>
    )
  }

  // Has an expiry date but it's still more than 30 days away — show quietly
  return (
    <Badge variant="secondary" className="text-xs">
      Gültig bis: {dateStr}
    </Badge>
  )
}
