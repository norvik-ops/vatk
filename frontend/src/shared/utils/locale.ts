import i18n from 'i18next'

/**
 * formatLocale — non-Hook-Variante für `Intl.*`-APIs aus statischem Code
 * (Helper-Funktionen, Toolbar-Strings, ScopedBlocks ohne React-Context).
 *
 * Sprint 16 / S16-10 — die Bulk-Migration der ~60 hardcoded `'de-DE'`-Stellen
 * aus dem Code. Wo `useFormatDate` (der Hook) verfügbar wäre, sollte er
 * bevorzugt werden — der re-rendert die Komponente bei Locale-Wechsel.
 * `formatLocale()` liest die aktuelle i18next-Locale einmalig zum Aufruf-
 * Zeitpunkt, was für die meisten Render-Pfade reicht.
 *
 * BCP47-Mapping ist identisch zum useFormatDate-Hook:
 *   de → de-DE, en → en-US, fr → fr-FR, nl → nl-NL
 */
const BCP47: Record<string, string> = {
  de: 'de-DE',
  en: 'en-US',
  fr: 'fr-FR',
  nl: 'nl-NL',
}

export function formatLocale(): string {
  const lang = (i18n.language || 'de').toLowerCase().split('-')[0]
  return BCP47[lang] ?? 'de-DE'
}
