import { useEffect } from 'react'
import { useThemeStore } from '../stores/theme'
import type { Theme } from '../stores/theme'

export type { Theme }

/**
 * Thin wrapper over `useThemeStore` that also wires up the system-theme
 * media query listener so the `system` mode reacts to OS-level changes.
 */
export function useTheme() {
  const { theme, setTheme, toggle } = useThemeStore()

  // Keep `system` mode in sync when OS preference changes
  useEffect(() => {
    if (theme !== 'system') return
    const mql = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => {
      const isDark = mql.matches
      document.documentElement.classList.toggle('dark', isDark)
    }
    mql.addEventListener('change', handler)
    return () => { mql.removeEventListener('change', handler); }
  }, [theme])

  function cycleTheme() {
    toggle()
  }

  return { theme, setTheme, cycleTheme }
}
