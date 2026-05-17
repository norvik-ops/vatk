import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { apiFetch } from '../api/client'
import {
  useFieldValidation,
  required,
  passwordStrength as passwordStrengthRule,
  getPasswordStrengthScore,
} from '../shared/hooks/useFieldValidation'
import { FieldError } from '../shared/components/FieldError'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'

const STRENGTH_COLORS = ['bg-red-500', 'bg-orange-500', 'bg-yellow-500', 'bg-green-500']
const STRENGTH_LABELS = ['Sehr schwach', 'Schwach', 'Mittel', 'Stark']

function PasswordStrengthBar({ password }: { password: string }) {
  const score = getPasswordStrengthScore(password)
  if (!password) return null

  return (
    <div className="mt-1.5 space-y-1">
      <div className="flex gap-1" role="img" aria-label={`Passwortstärke: ${STRENGTH_LABELS[score - 1] ?? 'Sehr schwach'}`}>
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className={`h-1 flex-1 rounded-full transition-colors duration-200 ${
              i < score ? STRENGTH_COLORS[score - 1] : 'bg-border'
            }`}
          />
        ))}
      </div>
      {score > 0 && (
        <p className="text-[11px] text-secondary">{STRENGTH_LABELS[score - 1]}</p>
      )}
    </div>
  )
}

export default function ResetPasswordPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const [password, setPassword] = useState('')
  const [passwordConfirm, setPasswordConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const passwordValidation = useFieldValidation(password, [required, passwordStrengthRule])
  const confirmValidation = useFieldValidation(passwordConfirm, [
    required,
    { test: (v) => v === password, message: 'Die Passwörter stimmen nicht überein.' },
  ])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)

    if (!passwordStrengthRule.test(password)) {
      setError('Das Passwort erfüllt die Sicherheitsanforderungen nicht.')
      return
    }
    if (password !== passwordConfirm) {
      setError('Die Passwörter stimmen nicht überein.')
      return
    }

    setLoading(true)
    try {
      await apiFetch('/auth/password-reset/confirm', {
        method: 'POST',
        body: JSON.stringify({ token, password }),
      })
      navigate('/login', {
        state: { successMessage: 'Passwort erfolgreich zurückgesetzt. Sie können sich jetzt anmelden.' },
        replace: true,
      })
    } catch {
      setError('Link ungültig oder abgelaufen. Bitte fordern Sie einen neuen Link an.')
    } finally {
      setLoading(false)
    }
  }

  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-bg p-4">
        <div className="w-full max-w-sm">
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2.5 mb-2">
                <img src="/logo.svg" alt="Vakt" className="w-9 h-9 shrink-0" />
                <span className="font-semibold text-[16px] text-brand">Vakt</span>
              </div>
              <CardTitle>Link ungültig</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-secondary">
                Dieser Link zum Zurücksetzen des Passworts ist ungültig oder abgelaufen.
              </p>
              <Link
                to="/auth/forgot-password"
                className="text-sm text-brand hover:underline block text-center"
              >
                Neuen Link anfordern
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg p-4">
      <div className="w-full max-w-sm space-y-4">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2.5 mb-2">
              <img src="/logo.svg" alt="Vakt" className="w-9 h-9 shrink-0" />
              <span className="font-semibold text-[16px] text-brand">Vakt</span>
            </div>
            <CardTitle>Neues Passwort festlegen</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => { void handleSubmit(e) }} className="space-y-4">
              <div className="space-y-1">
                <Label htmlFor="password">Neues Passwort</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Mindestens 10 Zeichen"
                  required
                  autoFocus
                  aria-invalid={!!passwordValidation.error}
                />
                <PasswordStrengthBar password={password} />
                <FieldError error={passwordValidation.error} />
              </div>
              <div className="space-y-1">
                <Label htmlFor="password-confirm">Passwort bestätigen</Label>
                <Input
                  id="password-confirm"
                  type="password"
                  value={passwordConfirm}
                  onChange={(e) => setPasswordConfirm(e.target.value)}
                  placeholder="Passwort wiederholen"
                  required
                  aria-invalid={!!confirmValidation.error}
                />
                <FieldError error={confirmValidation.error} />
              </div>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? 'Wird gespeichert…' : 'Passwort zurücksetzen'}
              </Button>
              <div className="text-center">
                <Link to="/login" className="text-sm text-secondary hover:text-primary hover:underline">
                  Zurück zum Login
                </Link>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
