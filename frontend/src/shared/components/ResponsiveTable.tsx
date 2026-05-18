import { useState, useEffect } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../components/ui/table'
import { cn } from '../../lib/utils'

// ── useMediaQuery ─────────────────────────────────────────────────────────────

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => window.matchMedia(query).matches)
  useEffect(() => {
    const mql = window.matchMedia(query)
    const handler = (e: MediaQueryListEvent) => setMatches(e.matches)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [query])
  return matches
}

// ── Column definition ─────────────────────────────────────────────────────────

export interface Column<T> {
  key: keyof T | string
  label: string
  render?: (row: T) => React.ReactNode
  /** If true, this column is omitted in the mobile card view */
  mobileHide?: boolean
  /** If true, this column is used as the card title (first non-hidden column by default) */
  mobileTitle?: boolean
}

export interface ResponsiveTableProps<T> {
  columns: Column<T>[]
  data: T[]
  onRowClick?: (row: T) => void
  /** The field used as React key (must be unique) */
  keyField: keyof T
}

function getCellValue<T>(row: T, key: keyof T | string): React.ReactNode {
  if (typeof key === 'string' && key.includes('.')) {
    // simple dot-path access
    return key.split('.').reduce<unknown>((obj, part) => {
      if (obj != null && typeof obj === 'object') return (obj as Record<string, unknown>)[part]
      return undefined
    }, row) as React.ReactNode
  }
  return row[key as keyof T] as React.ReactNode
}

// ── Desktop table ─────────────────────────────────────────────────────────────

function DesktopTable<T>({ columns, data, onRowClick, keyField }: ResponsiveTableProps<T>) {
  return (
    <div className="rounded-md border border-border bg-surface overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead key={String(col.key)}>{col.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => (
            <TableRow
              key={String(row[keyField])}
              className={cn(onRowClick && 'cursor-pointer hover:bg-surface2')}
              onClick={() => onRowClick?.(row)}
            >
              {columns.map((col) => (
                <TableCell key={String(col.key)}>
                  {col.render ? col.render(row) : String(getCellValue(row, col.key) ?? '—')}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

// ── Mobile card list ──────────────────────────────────────────────────────────

function MobileCardList<T>({ columns, data, onRowClick, keyField }: ResponsiveTableProps<T>) {
  const visibleColumns = columns.filter((col) => !col.mobileHide)
  const titleCol = visibleColumns.find((col) => col.mobileTitle) ?? visibleColumns[0]
  const bodyColumns = visibleColumns.filter((col) => col !== titleCol)

  return (
    <div className="space-y-3">
      {data.map((row) => (
        <div
          key={String(row[keyField])}
          className={cn(
            'rounded-lg border border-border bg-surface p-4 space-y-2',
            onRowClick && 'cursor-pointer hover:border-brand/40 transition-colors',
          )}
          onClick={() => onRowClick?.(row)}
        >
          {/* Title row */}
          <p className="font-semibold text-base text-primary leading-snug">
            {titleCol.render ? titleCol.render(row) : String(getCellValue(row, titleCol.key) ?? '—')}
          </p>

          {/* Key-value pairs */}
          <div className="space-y-1">
            {bodyColumns.map((col) => (
              <div key={String(col.key)} className="flex items-center gap-2 text-sm">
                <span className="text-secondary shrink-0 w-28 truncate">{col.label}:</span>
                <span className="text-primary">
                  {col.render ? col.render(row) : String(getCellValue(row, col.key) ?? '—')}
                </span>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

// ── Public component ──────────────────────────────────────────────────────────

export function ResponsiveTable<T>(props: ResponsiveTableProps<T>) {
  const isDesktop = useMediaQuery('(min-width: 1024px)')
  return isDesktop ? <DesktopTable {...props} /> : <MobileCardList {...props} />
}
