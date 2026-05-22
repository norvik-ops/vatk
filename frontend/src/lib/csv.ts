/**
 * Client-side CSV export utility.
 * Converts an array of plain objects to a CSV file and triggers a browser download.
 */
export function exportToCSV(filename: string, rows: Record<string, unknown>[]): void {
  if (rows.length === 0) return

  const headers = Object.keys(rows[0])

  function escapeCell(value: unknown): string {
    if (value === null || value === undefined) return ''
    let str: string
    if (typeof value === 'object' || typeof value === 'function') {
      str = JSON.stringify(value)
    } else {
      // value is string | number | boolean | symbol | bigint here
      str = (value as { toString(): string }).toString()
    }
    // Wrap in quotes if the value contains a comma, quote, or newline
    if (str.includes(',') || str.includes('"') || str.includes('\n')) {
      return `"${str.replace(/"/g, '""')}"`
    }
    return str
  }

  const csvLines: string[] = [
    headers.map(escapeCell).join(','),
    ...rows.map((row) => headers.map((h) => escapeCell(row[h])).join(',')),
  ]

  const blob = new Blob([csvLines.join('\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename.endsWith('.csv') ? filename : `${filename}.csv`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
