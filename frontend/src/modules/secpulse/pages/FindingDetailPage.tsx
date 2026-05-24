import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { Spinner } from '../../../components/Spinner'
import { ArrowLeft, Lightbulb, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../../../shared/components/PageHeader'
import { Breadcrumbs } from '../../../shared/components/Breadcrumbs'
import { trackPage } from '../../../shared/hooks/useRecentPages'
import { Button } from '../../../components/ui/button'
import { Badge } from '../../../components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '../../../components/ui/card'
import { Label } from '../../../components/ui/label'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../../../components/ui/select'
import { useFinding, usePatchFinding } from '../hooks/useFindings'
import type { Finding } from '../types'
import { cn } from '../../../lib/utils'
import { findingSeverityClass } from '../../../lib/statusMapping'
import { Comments } from '../../../shared/components/Comments'
import { useFormatDate } from '../../../shared/hooks/useFormatDate'
import { useAIInsights, useDismissInsight } from '../../secvitals/hooks/useAIInsights'

const severityClass = findingSeverityClass

function EvidenceSuggestionBanner({ findingId }: { findingId: string }) {
  const { data } = useAIInsights()
  const dismiss = useDismissInsight()
  const suggestions = data?.items.filter(
    (i) => i.type === 'evidence_suggestion' && i.finding_id === findingId
  ) ?? []

  if (suggestions.length === 0) return null

  return (
    <div className="px-6 pt-4 space-y-2">
      {suggestions.map((insight) => (
        <div key={insight.id} className="flex items-start gap-3 rounded-lg border border-brand/30 bg-brand/5 px-4 py-3">
          <Lightbulb className="w-4 h-4 mt-0.5 shrink-0 text-brand" />
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium text-primary">{insight.title}</p>
            <p className="text-xs text-secondary mt-0.5">{insight.message}</p>
          </div>
          <button
            onClick={() => { dismiss.mutate(insight.id); }}
            disabled={dismiss.isPending}
            className="shrink-0 text-muted-foreground hover:text-primary transition-colors"
            aria-label="Hinweis verwerfen"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      ))}
    </div>
  )
}

export default function FindingDetailPage() {
  const { t } = useTranslation()
  const { formatDate } = useFormatDate()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: finding, isLoading, error } = useFinding(id ?? '')
  const patch = usePatchFinding(id ?? '')

  const [status, setStatus] = useState<Finding['status'] | ''>('')
  const [notes, setNotes] = useState('')
  const [saved, setSaved] = useState(false)
  const [savedResolved, setSavedResolved] = useState(false)

  useEffect(() => {
    if (!saved) return
    const id = setTimeout(() => { setSaved(false); }, 2000)
    return () => { clearTimeout(id); }
  }, [saved])

  useEffect(() => {
    if (finding) trackPage(`/secpulse/findings/${id}`, finding.title, '🐛')
  }, [finding?.id])

  function currentStatus(): Finding['status'] | '' {
    return status || finding?.status || ''
  }

  async function handleSave() {
    if (!id) return
    const willResolve = (status || finding?.status) === 'resolved'
    await patch.mutateAsync({
      ...(status ? { status: status } : {}),
      notes: notes || undefined,
    })
    setSaved(true)
    if (willResolve) setSavedResolved(true)
  }

  if (isLoading) return (
    <div className="flex justify-center py-16">
      <Spinner size="md" />
    </div>
  )

  if (error || !finding) return (
    <div className="p-6">
      <p className="text-sm text-red-600">{error?.message ?? t('secpulse.findingDetail.notFound')}</p>
      <Button variant="outline" className="mt-4" onClick={() => { navigate('/secpulse/findings'); }}>
        <ArrowLeft className="w-4 h-4 mr-1" />{t('secpulse.findingDetail.back')}
      </Button>
    </div>
  )

  return (
    <div className="flex flex-col h-full">
      <Breadcrumbs items={[
        { label: 'Vakt Scan', href: '/secpulse' },
        { label: 'Findings', href: '/secpulse/findings' },
        { label: finding.title },
      ]} />
      <PageHeader
        title={finding.title}
        actions={
          <Button variant="outline" onClick={() => { navigate('/secpulse/findings'); }}>
            <ArrowLeft className="w-4 h-4 mr-1" />{t('secpulse.findingDetail.back')}
          </Button>
        }
      />

      <EvidenceSuggestionBanner findingId={finding.id} />

      <div className="p-6 grid grid-cols-3 gap-6">
        <div className="col-span-2 space-y-6">
          <Card>
            <CardHeader><CardTitle>{t('secpulse.findingDetail.description')}</CardTitle></CardHeader>
            <CardContent>
              <p className="text-sm text-secondary whitespace-pre-wrap">{finding.description}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader><CardTitle>{t('secpulse.findingDetail.updateStatus')}</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-1">
                <Label>{t('secpulse.findingDetail.status')}</Label>
                <Select
                  value={currentStatus()}
                  onValueChange={(v) => { setStatus(v as Finding['status']); }}
                >
                  <SelectTrigger className="w-48">
                    <SelectValue placeholder={t('secpulse.findingDetail.statusPlaceholder')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="open">{t('secpulse.status.open')}</SelectItem>
                    <SelectItem value="in_progress">{t('secpulse.status.in_progress')}</SelectItem>
                    <SelectItem value="accepted_risk">{t('secpulse.status.accepted_risk')}</SelectItem>
                    <SelectItem value="false_positive">{t('secpulse.status.false_positive')}</SelectItem>
                    <SelectItem value="resolved">{t('secpulse.status.resolved')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1">
                <Label htmlFor="notes">{t('secpulse.findingDetail.notes')}</Label>
                <textarea
                  id="notes"
                  rows={4}
                  className="w-full rounded-md border border-border px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-brand"
                  placeholder={t('secpulse.findingDetail.notesPlaceholder')}
                  defaultValue={finding.notes ?? ''}
                  onChange={(e) => { setNotes(e.target.value); }}
                />
              </div>

              {patch.isError && <p className="text-sm text-red-600">{patch.error.message}</p>}
              {saved && !savedResolved && <p className="text-sm text-green-600">{t('secpulse.findingDetail.saved')}</p>}
              {savedResolved && (
                <p className="text-sm text-green-600">
                  {t('secpulse.findingDetail.saved')} —{' '}
                  <Link to="/secvitals/evidence/auto" className="underline">
                    Finding-Auflösung als Evidence in Vakt Comply gespeichert
                  </Link>
                </p>
              )}

              <Button onClick={() => { void handleSave() }} disabled={patch.isPending}>
                {patch.isPending ? t('secpulse.findingDetail.saving') : t('secpulse.findingDetail.saveChanges')}
              </Button>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-4">
          <Card>
            <CardHeader><CardTitle>{t('secpulse.findingDetail.details')}</CardTitle></CardHeader>
            <CardContent>
              <dl className="space-y-3 text-sm">
                <div>
                  <dt className="text-secondary">{t('secpulse.findingDetail.severity')}</dt>
                  <dd className="mt-0.5">
                    <Badge className={cn('capitalize', severityClass[finding.severity])}>{finding.severity}</Badge>
                  </dd>
                </div>
                <div>
                  <dt className="text-secondary">{t('secpulse.findingDetail.status')}</dt>
                  <dd className="mt-0.5 capitalize text-primary">{finding.status.replace(/_/g, ' ')}</dd>
                </div>
                {finding.cve_id && (
                  <div>
                    <dt className="text-secondary">CVE</dt>
                    <dd className="mt-0.5 font-mono text-xs text-primary">{finding.cve_id}</dd>
                  </div>
                )}
                {finding.cvss_score != null && (
                  <div>
                    <dt className="text-secondary">CVSS Score</dt>
                    <dd className="mt-0.5 text-primary font-semibold">{finding.cvss_score.toFixed(1)}</dd>
                  </div>
                )}
                <div>
                  <dt className="text-secondary">{t('secpulse.findingDetail.discovered')}</dt>
                  <dd className="mt-0.5 text-primary">{formatDate(finding.created_at)}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Comments */}
      <div className="px-6 pb-6">
        <Comments entityType="finding" entityId={finding.id} />
      </div>
    </div>
  )
}
