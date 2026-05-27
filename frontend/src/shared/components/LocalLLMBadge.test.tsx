import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { LocalLLMBadge } from './LocalLLMBadge'

// react-i18next is wrapped here so the component renders deterministic
// English-ish strings regardless of the locale config. The badge renders one
// of two i18n keys — testing them by *key* keeps the test independent of the
// actual translation files.
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (opts && 'model' in opts) {
        return `${key}::${opts.model as string}`
      }
      return key
    },
  }),
}))

describe('LocalLLMBadge — providerHost gate', () => {
  // Audit finding F2 / ADR-0034: when providerHost is omitted the badge used
  // to fall through to "lokal" — a trust-cue lie when the backend is pointed
  // at a cloud provider. The badge is intentionally permissive on local-host
  // names but only flips to "lokal" when it positively recognises one.

  it('renders LOCAL when providerHost is a known local container name', () => {
    render(<LocalLLMBadge providerHost="ollama" model="qwen2.5:3b" />)
    expect(screen.getByText(/ai\.localBadge\.local/)).toBeInTheDocument()
  })

  it('renders LOCAL when providerHost contains "ollama" (subdomain)', () => {
    render(<LocalLLMBadge providerHost="my-ollama.internal" model="qwen" />)
    expect(screen.getByText(/ai\.localBadge\.local/)).toBeInTheDocument()
  })

  it('renders CLOUD when providerHost is api.openai.com', () => {
    render(<LocalLLMBadge providerHost="api.openai.com" model="gpt-4" />)
    expect(screen.getByText(/ai\.localBadge\.cloud/)).toBeInTheDocument()
    expect(screen.queryByText(/ai\.localBadge\.local/)).not.toBeInTheDocument()
  })

  it('renders CLOUD for mistral.ai', () => {
    render(<LocalLLMBadge providerHost="api.mistral.ai" model="mistral-large" />)
    expect(screen.getByText(/ai\.localBadge\.cloud/)).toBeInTheDocument()
  })

  it('renders CLOUD for groq.com (non-local)', () => {
    render(<LocalLLMBadge providerHost="api.groq.com" />)
    expect(screen.getByText(/ai\.localBadge\.cloud/)).toBeInTheDocument()
  })

  // The component still has a "no info → lokal" fallback. The fix on the
  // *caller* side (SecVitalsOverviewPage) is what closes the trust hole —
  // this test pins the current badge behaviour so any future change is
  // intentional. See docs/adr/0034-localllmbadge-provider-host.md.
  it('falls back to LOCAL when providerHost is undefined (legacy callers)', () => {
    render(<LocalLLMBadge model="unknown" />)
    expect(screen.getByText(/ai\.localBadge\.local/)).toBeInTheDocument()
  })
})
