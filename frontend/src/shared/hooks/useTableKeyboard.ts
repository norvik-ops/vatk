import { useState } from 'react'

/**
 * Keyboard navigation hook for table rows.
 * ArrowDown / Tab → next row, ArrowUp → previous row, Enter → onSelect(index).
 */
export function useTableKeyboard(rowCount: number, onSelect: (idx: number) => void) {
  const [focusIdx, setFocusIdx] = useState(-1)

  function onKeyDown(e: React.KeyboardEvent, currentIdx: number) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setFocusIdx(Math.min(currentIdx + 1, rowCount - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setFocusIdx(Math.max(currentIdx - 1, 0))
    } else if (e.key === 'Enter') {
      onSelect(currentIdx)
    }
  }

  return { focusIdx, setFocusIdx, onKeyDown }
}
