import { describe, it, expect } from 'vitest'
import deLocale from '../../../i18n/locales/de.json'
import enLocale from '../../../i18n/locales/en.json'
import frLocale from '../../../i18n/locales/fr.json'
import nlLocale from '../../../i18n/locales/nl.json'

// Sprint-59 i18n contract test for AccessReviewsPage. The page references
// these keys via the `t()` helper; if any key is removed from one locale
// the runtime fallback would silently swap in the key path as visible UI
// text. We pin the contract here so the regression bites in CI, not in
// production.
const requiredPaths = [
  ['secvitals', 'accessReviews', 'title'],
  ['secvitals', 'accessReviews', 'description'],
  ['secvitals', 'accessReviews', 'emptyTitle'],
  ['secvitals', 'accessReviews', 'addCampaign'],
  ['secvitals', 'accessReviews', 'editCampaign'],
  ['secvitals', 'accessReviews', 'fields', 'title'],
  ['secvitals', 'accessReviews', 'fields', 'description'],
  ['secvitals', 'accessReviews', 'fields', 'reviewerEmail'],
  ['secvitals', 'accessReviews', 'fields', 'scope'],
  ['secvitals', 'accessReviews', 'fields', 'dueDate'],
  ['secvitals', 'accessReviews', 'fields', 'status'],
  ['secvitals', 'accessReviews', 'fields', 'user'],
  ['secvitals', 'accessReviews', 'fields', 'userEmail'],
  ['secvitals', 'accessReviews', 'fields', 'role'],
  ['secvitals', 'accessReviews', 'fields', 'decision'],
  ['secvitals', 'accessReviews', 'fields', 'comment'],
  ['secvitals', 'accessReviews', 'fields', 'actions'],
  ['secvitals', 'accessReviews', 'placeholders', 'userEmail'],
  ['secvitals', 'accessReviews', 'placeholders', 'role'],
  ['secvitals', 'accessReviews', 'placeholders', 'title'],
  ['secvitals', 'accessReviews', 'placeholders', 'description'],
  ['secvitals', 'accessReviews', 'placeholders', 'reviewerEmail'],
  ['secvitals', 'accessReviews', 'placeholders', 'scope'],
  ['secvitals', 'accessReviews', 'status', 'draft'],
  ['secvitals', 'accessReviews', 'status', 'active'],
  ['secvitals', 'accessReviews', 'status', 'completed'],
  ['secvitals', 'accessReviews', 'status', 'cancelled'],
  ['secvitals', 'accessReviews', 'decision', 'pending'],
  ['secvitals', 'accessReviews', 'decision', 'approved'],
  ['secvitals', 'accessReviews', 'decision', 'revoked'],
]

const requiredAISystemsPaths = [
  ['secvitals', 'aiSystems', 'title'],
  ['secvitals', 'aiSystems', 'description'],
  ['secvitals', 'aiSystems', 'emptyTitle'],
  ['secvitals', 'aiSystems', 'add'],
  ['secvitals', 'aiSystems', 'edit'],
  ['secvitals', 'aiSystems', 'filterAll'],
  ['secvitals', 'aiSystems', 'actions', 'classify'],
  ['secvitals', 'aiSystems', 'actions', 'documentation'],
  ['secvitals', 'aiSystems', 'fields', 'name'],
  ['secvitals', 'aiSystems', 'fields', 'provider'],
  ['secvitals', 'aiSystems', 'fields', 'useCase'],
  ['secvitals', 'aiSystems', 'fields', 'description'],
  ['secvitals', 'aiSystems', 'fields', 'affectedGroups'],
  ['secvitals', 'aiSystems', 'fields', 'autonomy'],
  ['secvitals', 'aiSystems', 'fields', 'riskClass'],
  ['secvitals', 'aiSystems', 'fields', 'status'],
  ['secvitals', 'aiSystems', 'fields', 'classification'],
  ['secvitals', 'aiSystems', 'fields', 'classifiedBy'],
  ['secvitals', 'aiSystems', 'autonomyLevel', 'assistive'],
  ['secvitals', 'aiSystems', 'autonomyLevel', 'semiAutonomous'],
  ['secvitals', 'aiSystems', 'autonomyLevel', 'fullyAutonomous'],
  ['secvitals', 'aiSystems', 'riskClassLevel', 'minimal'],
  ['secvitals', 'aiSystems', 'riskClassLevel', 'limited'],
  ['secvitals', 'aiSystems', 'riskClassLevel', 'high'],
  ['secvitals', 'aiSystems', 'riskClassLevel', 'unacceptable'],
  ['secvitals', 'aiSystems', 'riskClassLevel', 'prohibited'],
  ['secvitals', 'aiSystems', 'statusLevel', 'classified'],
  ['secvitals', 'aiSystems', 'statusLevel', 'approved'],
  ['secvitals', 'aiSystems', 'statusLevel', 'compliant'],
  ['secvitals', 'aiSystems', 'statusLevel', 'decommissioned'],
]

function get(obj: unknown, path: string[]): string | undefined {
  let cur: unknown = obj
  for (const p of path) {
    if (cur && typeof cur === 'object' && p in (cur as Record<string, unknown>)) {
      cur = (cur as Record<string, unknown>)[p]
    } else {
      return undefined
    }
  }
  return typeof cur === 'string' ? cur : undefined
}

const locales: Record<string, unknown> = {
  de: deLocale,
  en: enLocale,
  fr: frLocale,
  nl: nlLocale,
}

describe('secvitals i18n contract — AccessReviewsPage', () => {
  for (const lang of Object.keys(locales)) {
    for (const path of requiredPaths) {
      it(`${lang}: ${path.join('.')} is a non-empty string`, () => {
        const value = get(locales[lang], path)
        expect(value).toBeDefined()
        expect(value).not.toBe('')
      })
    }
  }
})

describe('secvitals i18n contract — AISystemsPage', () => {
  for (const lang of Object.keys(locales)) {
    for (const path of requiredAISystemsPaths) {
      it(`${lang}: ${path.join('.')} is a non-empty string`, () => {
        const value = get(locales[lang], path)
        expect(value).toBeDefined()
        expect(value).not.toBe('')
      })
    }
  }
})
