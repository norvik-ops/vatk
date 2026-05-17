import { useState, useMemo } from 'react'

export type SortDir = 'asc' | 'desc'

export interface SortableTableResult<T> {
  sorted: T[]
  sortKey: keyof T | null
  sortDir: SortDir
  toggleSort: (key: keyof T) => void
}

export function useSortableTable<T>(
  data: T[],
  defaultSort?: { key: keyof T; dir: SortDir },
): SortableTableResult<T> {
  const [sortKey, setSortKey] = useState<keyof T | null>(defaultSort?.key ?? null)
  const [sortDir, setSortDir] = useState<SortDir>(defaultSort?.dir ?? 'asc')

  function toggleSort(key: keyof T) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  const sorted = useMemo(() => {
    if (!sortKey) return data
    return [...data].sort((a, b) => {
      const av = a[sortKey]
      const bv = b[sortKey]

      // Null/undefined always last
      if (av == null && bv == null) return 0
      if (av == null) return 1
      if (bv == null) return -1

      let cmp: number
      if (typeof av === 'number' && typeof bv === 'number') {
        cmp = av - bv
      } else {
        cmp = String(av).localeCompare(String(bv), 'de', { sensitivity: 'base' })
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [data, sortKey, sortDir])

  return { sorted, sortKey, sortDir, toggleSort }
}
