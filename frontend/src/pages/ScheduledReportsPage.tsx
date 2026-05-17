/**
 * ScheduledReportsPage — geplante Berichte verwalten.
 *
 * TODO: Backend-Anbindung über /api/v1/reports/scheduled sobald Endpoint verfügbar.
 *       Aktuell: lokaler State (useState) als vollständige UI-Implementierung.
 */
import { useState } from 'react'
import { Plus, Pencil, Trash2, Play, Calendar, X } from 'lucide-react'
import { PageHeader } from '../shared/components/PageHeader'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Badge } from '../components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select'
import { toast } from '../shared/hooks/useToast'

// ─── Types ────────────────────────────────────────────────────────────────────

type ReportType = 'compliance' | 'findings' | 'risk'
type Schedule = 'weekly' | 'monthly' | 'quarterly'
type Format = 'pdf' | 'csv'

interface ScheduledReport {
  id: string
  name: string
  type: ReportType
  schedule: Schedule
  recipients: string[]
  format: Format
  next_run: string
  created_at: string
}

// ─── Labels ───────────────────────────────────────────────────────────────────

const REPORT_TYPE_LABELS: Record<ReportType, string> = {
  compliance: 'Compliance-Übersicht',
  findings:   'Findings-Report',
  risk:       'Risk-Report',
}

const SCHEDULE_LABELS: Record<Schedule, string> = {
  weekly:    'Wöchentlich (Montag)',
  monthly:   'Monatlich (1. des Monats)',
  quarterly: 'Vierteljährlich',
}

const FORMAT_LABELS: Record<Format, string> = {
  pdf: 'PDF',
  csv: 'CSV',
}

// ─── Next run helpers ─────────────────────────────────────────────────────────

function computeNextRun(schedule: Schedule): string {
  const now = new Date()
  if (schedule === 'weekly') {
    const next = new Date(now)
    const daysUntilMonday = (1 - now.getDay() + 7) % 7 || 7
    next.setDate(now.getDate() + daysUntilMonday)
    return next.toISOString()
  }
  if (schedule === 'monthly') {
    const next = new Date(now.getFullYear(), now.getMonth() + 1, 1)
    return next.toISOString()
  }
  // quarterly
  const currentQ = Math.floor(now.getMonth() / 3)
  const nextQMonth = (currentQ + 1) * 3
  const next = new Date(now.getFullYear(), nextQMonth, 1)
  return next.toISOString()
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

// ─── Chips input for email addresses ──────────────────────────────────────────

interface ChipsInputProps {
  value: string[]
  onChange: (v: string[]) => void
}

function ChipsInput({ value, onChange }: ChipsInputProps) {
  const [input, setInput] = useState('')

  function addChip() {
    const trimmed = input.trim()
    if (!trimmed || value.includes(trimmed)) { setInput(''); return }
    onChange([...value, trimmed])
    setInput('')
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addChip()
    }
    if (e.key === 'Backspace' && !input && value.length > 0) {
      onChange(value.slice(0, -1))
    }
  }

  function removeChip(chip: string) {
    onChange(value.filter((v) => v !== chip))
  }

  return (
    <div className="min-h-[38px] border border-border rounded-md px-2 py-1 flex flex-wrap gap-1 focus-within:ring-2 focus-within:ring-brand/30 bg-background">
      {value.map((chip) => (
        <span
          key={chip}
          className="flex items-center gap-1 bg-brand/10 text-brand text-xs px-2 py-0.5 rounded-full"
        >
          {chip}
          <button
            type="button"
            onClick={() => removeChip(chip)}
            className="hover:text-red-500 transition-colors"
            aria-label={`${chip} entfernen`}
          >
            <X className="w-3 h-3" />
          </button>
        </span>
      ))}
      <input
        type="email"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={addChip}
        placeholder={value.length === 0 ? 'E-Mail eingeben, Enter drücken' : ''}
        className="flex-1 min-w-[160px] text-sm outline-none bg-transparent placeholder:text-muted-foreground"
      />
    </div>
  )
}

// ─── Report Form Dialog ───────────────────────────────────────────────────────

interface ReportDialogProps {
  open: boolean
  onClose: () => void
  initial?: ScheduledReport
  onSave: (data: Omit<ScheduledReport, 'id' | 'created_at' | 'next_run'>) => void
}

const emptyForm = {
  name: '',
  type: 'compliance' as ReportType,
  schedule: 'monthly' as Schedule,
  recipients: [] as string[],
  format: 'pdf' as Format,
}

function ReportDialog({ open, onClose, initial, onSave }: ReportDialogProps) {
  const [form, setForm] = useState(() =>
    initial
      ? { name: initial.name, type: initial.type, schedule: initial.schedule, recipients: initial.recipients, format: initial.format }
      : { ...emptyForm }
  )

  function handleSave() {
    if (!form.name.trim() || form.recipients.length === 0) return
    onSave(form)
    onClose()
  }

  function handleOpenChange(v: boolean) {
    if (!v) onClose()
  }

  const isEdit = !!initial
  const canSave = form.name.trim() !== '' && form.recipients.length > 0

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Bericht bearbeiten' : 'Bericht planen'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Name */}
          <div className="space-y-1.5">
            <Label htmlFor="rep-name">Name des Berichts</Label>
            <Input
              id="rep-name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="z.B. Monatlicher Compliance-Bericht"
            />
          </div>

          {/* Type */}
          <div className="space-y-1.5">
            <Label>Typ</Label>
            <Select
              value={form.type}
              onValueChange={(v) => setForm({ ...form, type: v as ReportType })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(Object.entries(REPORT_TYPE_LABELS) as [ReportType, string][]).map(([v, l]) => (
                  <SelectItem key={v} value={v}>{l}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Schedule */}
          <div className="space-y-1.5">
            <Label>Zeitplan</Label>
            <Select
              value={form.schedule}
              onValueChange={(v) => setForm({ ...form, schedule: v as Schedule })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(Object.entries(SCHEDULE_LABELS) as [Schedule, string][]).map(([v, l]) => (
                  <SelectItem key={v} value={v}>{l}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Recipients */}
          <div className="space-y-1.5">
            <Label>Empfänger</Label>
            <ChipsInput
              value={form.recipients}
              onChange={(v) => setForm({ ...form, recipients: v })}
            />
            <p className="text-[11px] text-secondary">
              E-Mail eingeben und Enter drücken. Mehrere Empfänger möglich.
            </p>
          </div>

          {/* Format */}
          <div className="space-y-1.5">
            <Label>Format</Label>
            <Select
              value={form.format}
              onValueChange={(v) => setForm({ ...form, format: v as Format })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(Object.entries(FORMAT_LABELS) as [Format, string][]).map(([v, l]) => (
                  <SelectItem key={v} value={v}>{l}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {(!form.name.trim() || form.recipients.length === 0) && (
            <p className="text-[11px] text-amber-600">
              Name und mindestens ein Empfänger sind erforderlich.
            </p>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Abbrechen</Button>
          <Button onClick={handleSave} disabled={!canSave}>Speichern</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ─── Report Card ──────────────────────────────────────────────────────────────

interface ReportCardProps {
  report: ScheduledReport
  onEdit: () => void
  onDelete: () => void
  onRunNow: () => void
}

function ReportCard({ report, onEdit, onDelete, onRunNow }: ReportCardProps) {
  return (
    <div className="bg-surface border border-border rounded-xl p-5 flex flex-col gap-3 hover:border-brand/30 transition-colors">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="font-semibold text-primary text-sm">{report.name}</h3>
          <p className="text-xs text-secondary mt-0.5">{REPORT_TYPE_LABELS[report.type]}</p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0"
            title="Jetzt senden"
            onClick={onRunNow}
          >
            <Play className="w-3.5 h-3.5" aria-hidden="true" />
            <span className="sr-only">Jetzt senden</span>
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0"
            title="Bearbeiten"
            onClick={onEdit}
          >
            <Pencil className="w-3.5 h-3.5" aria-hidden="true" />
            <span className="sr-only">Bearbeiten</span>
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 w-7 p-0 text-secondary hover:text-red-500 hover:bg-red-500/10"
            title="Löschen"
            onClick={onDelete}
          >
            <Trash2 className="w-3.5 h-3.5" aria-hidden="true" />
            <span className="sr-only">Löschen</span>
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
        <div className="text-secondary">Zeitplan</div>
        <div className="text-primary font-medium">{SCHEDULE_LABELS[report.schedule]}</div>

        <div className="text-secondary">Nächste Ausführung</div>
        <div className="text-primary font-medium">{formatDate(report.next_run)}</div>

        <div className="text-secondary">Format</div>
        <div>
          <Badge variant="secondary" className="text-[10px]">{FORMAT_LABELS[report.format]}</Badge>
        </div>

        <div className="text-secondary">Empfänger</div>
        <div className="flex flex-wrap gap-1">
          {report.recipients.map((r) => (
            <span key={r} className="text-[10px] bg-surface2 border border-border rounded px-1.5 py-0.5 text-primary">
              {r}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}

// ─── Empty State ──────────────────────────────────────────────────────────────

function EmptyReports({ onCreate }: { onCreate: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 gap-4">
      <div className="p-4 rounded-full bg-surface2">
        <Calendar className="w-8 h-8 text-secondary" aria-hidden="true" />
      </div>
      <div className="text-center">
        <p className="font-semibold text-primary">Noch keine geplanten Berichte</p>
        <p className="text-sm text-secondary mt-1 max-w-sm">
          Planen Sie regelmäßige Berichte, die automatisch per E-Mail versendet werden.
        </p>
      </div>
      <Button onClick={onCreate}>
        <Plus className="w-4 h-4 mr-1.5" />
        Bericht planen
      </Button>
    </div>
  )
}

// ─── Main Page ────────────────────────────────────────────────────────────────

// TODO: Replace local state with API calls once /api/v1/reports/scheduled is available.
// useQuery: queryKey ['scheduled-reports']
// Mutations: POST /api/v1/reports/scheduled, PUT /api/v1/reports/scheduled/:id,
//            DELETE /api/v1/reports/scheduled/:id, POST /api/v1/reports/scheduled/:id/run

export default function ScheduledReportsPage() {
  const [reports, setReports] = useState<ScheduledReport[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<ScheduledReport | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<ScheduledReport | null>(null)

  function openCreate() {
    setEditTarget(undefined)
    setDialogOpen(true)
  }

  function openEdit(r: ScheduledReport) {
    setEditTarget(r)
    setDialogOpen(true)
  }

  function handleSave(data: Omit<ScheduledReport, 'id' | 'created_at' | 'next_run'>) {
    if (editTarget) {
      setReports((prev) =>
        prev.map((r) =>
          r.id === editTarget.id
            ? { ...r, ...data, next_run: computeNextRun(data.schedule) }
            : r
        )
      )
      toast('Bericht aktualisiert', 'success')
    } else {
      const newReport: ScheduledReport = {
        id: crypto.randomUUID(),
        ...data,
        next_run: computeNextRun(data.schedule),
        created_at: new Date().toISOString(),
      }
      setReports((prev) => [...prev, newReport])
      toast('Bericht geplant', 'success')
    }
  }

  function handleDelete() {
    if (!deleteTarget) return
    setReports((prev) => prev.filter((r) => r.id !== deleteTarget.id))
    setDeleteTarget(null)
    toast('Bericht gelöscht', 'success')
  }

  function handleRunNow(report: ScheduledReport) {
    // TODO: POST /api/v1/reports/scheduled/:id/run
    toast(`„${report.name}" wurde zur sofortigen Ausführung eingeplant`, 'info')
  }

  return (
    <div className="flex flex-col h-full">
      <PageHeader
        title="Geplante Berichte"
        description="Automatisch versendete Compliance- und Sicherheitsberichte."
        actions={
          reports.length > 0 ? (
            <Button onClick={openCreate}>
              <Plus className="w-4 h-4 mr-1.5" />
              Bericht planen
            </Button>
          ) : undefined
        }
      />

      <div className="flex-1 p-6 overflow-auto">
        {reports.length === 0 ? (
          <EmptyReports onCreate={openCreate} />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 max-w-5xl">
            {reports.map((r) => (
              <ReportCard
                key={r.id}
                report={r}
                onEdit={() => openEdit(r)}
                onDelete={() => setDeleteTarget(r)}
                onRunNow={() => handleRunNow(r)}
              />
            ))}
          </div>
        )}

        <p className="text-[11px] text-secondary mt-6">
          Backend-Anbindung ausstehend — Daten werden aktuell nur im lokalen State gespeichert.
        </p>
      </div>

      {/* Create / Edit Dialog */}
      {dialogOpen && (
        <ReportDialog
          open={dialogOpen}
          onClose={() => { setDialogOpen(false); setEditTarget(undefined) }}
          initial={editTarget}
          onSave={handleSave}
        />
      )}

      {/* Delete Confirmation */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Bericht löschen?</AlertDialogTitle>
            <AlertDialogDescription>
              Der geplante Bericht <strong>{deleteTarget?.name}</strong> wird dauerhaft entfernt.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Abbrechen</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-red-600 hover:bg-red-700 focus:ring-red-600"
            >
              Löschen
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
