/**
 * Exportiert Compliance-Framework-Daten als RTF (Word-kompatibel).
 * RTF ist plaintext — kein npm-Package nötig.
 */

function escapeRtf(text: string): string {
  return text
    .replace(/\\/g, '\\\\')
    .replace(/\{/g, '\\{')
    .replace(/\}/g, '\\}')
    .replace(/[^\x00-\x7F]/g, (char) => {
      const code = char.charCodeAt(0)
      return `\\'${code.toString(16).padStart(2, '0')}`
    })
}

export function exportAsRTF(
  title: string,
  sections: { heading: string; rows: string[][] }[],
): void {
  const lines: string[] = [
    '{\\rtf1\\ansi\\ansicpg1252\\deff0',
    '{\\fonttbl{\\f0\\froman\\fcharset0 Times New Roman;}{\\f1\\fswiss\\fcharset0 Arial;}}',
    '{\\colortbl;\\red0\\green0\\blue0;\\red60\\green90\\blue180;}',
    '\\f1\\fs28\\b ' + escapeRtf(title) + '\\b0\\par\\par',
  ]

  for (const section of sections) {
    lines.push(`\\f1\\fs24\\b\\cf2 ${escapeRtf(section.heading)}\\b0\\cf1\\par`)
    for (const row of section.rows) {
      lines.push(`\\f1\\fs20 ${row.map(escapeRtf).join('  |  ')}\\par`)
    }
    lines.push('\\par')
  }

  lines.push('}')

  const rtf = lines.join('\n')
  const blob = new Blob([rtf], { type: 'application/rtf' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${title.replace(/[^a-zA-Z0-9]/g, '_')}.rtf`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
