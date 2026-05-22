import { ChevronUp, ChevronDown, ChevronsUpDown } from 'lucide-react'
import type { SortDir } from '../hooks/useSortableTable'

interface SortableHeaderProps<T> {
  label: string
  sortKey: T
  currentSortKey: T | null
  currentDir: SortDir
  onSort: (key: T) => void
  className?: string
}

export function SortableHeader<T>({
  label,
  sortKey,
  currentSortKey,
  currentDir,
  onSort,
  className,
}: SortableHeaderProps<T>) {
  const isActive = currentSortKey === sortKey

  return (
    <th
      scope="col"
      className={`cursor-pointer select-none whitespace-nowrap ${className ?? ''}`}
      onClick={() => { onSort(sortKey); }}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {isActive ? (
          currentDir === 'asc' ? (
            <ChevronUp className="w-3.5 h-3.5 text-brand" />
          ) : (
            <ChevronDown className="w-3.5 h-3.5 text-brand" />
          )
        ) : (
          <ChevronsUpDown className="w-3.5 h-3.5 text-secondary/60" />
        )}
      </span>
    </th>
  )
}
