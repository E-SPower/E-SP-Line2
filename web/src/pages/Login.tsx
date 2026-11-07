import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight, GitBranch, Languages, Loader2, Moon, ShieldCheck, Sun, Wifi } from 'lucide-react'
import apiClient from '../api/client'
import { Button } from '../components/ui/button'
import { FormField, controlClassName } from '../components/ui/form-field'
import { useTheme } from '../components/theme-provider'

export default function Login() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  // Set by the API client when a 401 bounced the reader here.
  const next = searchParams.get('next') || '/dashboard'
  const { resolvedTheme, toggleTheme } = useTheme()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const result = await apiClient.login(username, password)
      if (result.error) {
        setError(result.error)
      } else {
        navigate(next, { replace: true })
      }
    } catch {
      setError(t('auth.failedHint'))
    } finally {
      setLoading(false)
    }
  }

  const toggleLanguage = () => {
    void i18n.changeLanguage(i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN')
  }

  const features = [
    { icon: GitBranch, label: t('auth.featureRoutes') },
    { icon: Wifi, label: t('auth.featureMonitor') },
    { icon: Languages, label: t('auth.featureI18n') },
  ]

  return (
    <div className="grid min-h-screen bg-background lg:grid-cols-2">
      {/* Brand panel — the pure `primary` plane (black in light, white in dark). */}
      <aside className="relative hidden overflow-hidden border-r border-border bg-primary p-10 text-primary-foreground lg:flex lg:flex-col lg:justify-between">
        {/* Soft fluid geometry — geometry and timing live in index.css so the
            reduced-motion still is handled globally, not here. */}
        <div aria-hidden className="aurora">
          <span className="aurora-blob aurora-blob-a" />
          <span className="aurora-blob aurora-blob-b" />
          <span className="aurora-blob aurora-blob-c" />
        </div>

        <div className="relative flex items-center gap-2.5">
          {/* The mark is multi-colour artwork with white interiors, so it sits on
              a white plate in both modes: `--primary-foreground` is already the
              near-white plate in light, and dark needs it pinned so the blue
              does not land on the near-black panel. Border and radius are tokens. */}
          <span className="grid size-9 place-items-center rounded-md border border-primary-foreground/20 bg-primary-foreground dark:bg-white">
            {/* The adjacent title already names it, so a non-empty alt would
                make a screen reader say "E-SP-Line2" twice. */}
            <img src="/espl-logo.png" alt="" className="size-6 object-contain" />
          </span>
          <div>
            <p className="text-sm font-semibold tracking-tight">{t('app.title')}</p>
            <p className="text-xs text-primary-foreground/60">{t('app.subtitle')}</p>
          </div>
        </div>

        <div className="relative max-w-md">
          <h2 className="text-3xl font-semibold leading-tight tracking-tight">{t('auth.brandHeadline')}</h2>
          <p className="mt-3 text-sm leading-relaxed text-primary-foreground/70">{t('auth.brandDescription')}</p>

          <ul className="mt-8 space-y-2.5">
            {features.map((feature) => (
              <li key={feature.label} className="flex items-center gap-2.5 text-sm text-primary-foreground/80">
                <feature.icon aria-hidden className="size-4 shrink-0" />
                {feature.label}
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-primary-foreground/50">{t('app.version')}</p>
      </aside>

      {/* Form panel. `min-w-0` on the panel and the centred column: as a grid
          item the panel otherwise takes its max-content width, so one long
          word in a heading or field label widened the whole page. */}
      <main className="relative flex min-w-0 flex-col">
        <div aria-hidden className="panel-wash" />

        <div className="relative flex items-center justify-end gap-1 p-4">
          <Button variant="ghost" size="icon-sm" onClick={toggleLanguage} aria-label={t('common.switchLanguage')}>
            <Languages />
          </Button>
          <Button variant="ghost" size="icon-sm" onClick={toggleTheme} aria-label={t('common.toggleTheme')}>
            {resolvedTheme === 'dark' ? <Sun /> : <Moon />}
          </Button>
        </div>

        <div className="relative flex min-w-0 flex-1 items-center justify-center px-5 pb-16">
          <div className="w-full min-w-0 max-w-sm">
            {/* Mobile-only brand mark — the adjacent title is its label. */}
            <div className="mb-6 flex min-w-0 items-center gap-2.5 lg:hidden">
              <span aria-hidden className="grid size-9 shrink-0 place-items-center rounded-md border border-border bg-primary-foreground dark:bg-white">
                <img src="/espl-logo.png" alt="" className="size-6 object-contain" />
              </span>
              <div className="min-w-0">
                <p className="break-words text-sm font-semibold text-text-primary">{t('app.title')}</p>
                <p className="break-words text-xs text-text-tertiary">{t('app.subtitle')}</p>
              </div>
            </div>

            <h1 className="break-words text-2xl font-semibold tracking-tight text-text-primary">{t('auth.loginTitle')}</h1>
            <p className="mt-1 break-words text-sm text-text-tertiary">{t('auth.signInHint')}</p>

            <form onSubmit={handleSubmit} className="mt-6 space-y-4" noValidate>
              <FormField label={t('auth.username')} required>
                {(props) => (
                  <input
                    {...props}
                    name="username"
                    type="text"
                    autoComplete="username"
                    required
                    autoFocus
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    className={controlClassName}
                  />
                )}
              </FormField>

              <FormField label={t('auth.password')} required>
                {(props) => (
                  <input
                    {...props}
                    name="password"
                    type="password"
                    autoComplete="current-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={controlClassName}
                  />
                )}
              </FormField>

              {error && (
                <div role="alert" className="rounded-md border border-danger-border bg-danger-bg px-3 py-2">
                  <p className="text-xs font-semibold text-danger-fg">{t('auth.failed')}</p>
                  <p className="mt-0.5 text-xs text-danger-fg/90">{error}</p>
                </div>
              )}

              <Button type="submit" variant="primary" className="w-full" disabled={loading}>
                {loading ? (
                  <>
                    <Loader2 className="animate-spin" />
                    {t('common.loading')}
                  </>
                ) : (
                  <>
                    {t('auth.login')}
                    <ArrowRight />
                  </>
                )}
              </Button>
            </form>

            <p className="mt-4 break-words text-sm text-text-tertiary">
              {t('auth.noAccount')}{' '}
              <Link
                to="/register"
                className="rounded-xs font-medium text-text-primary underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring/50"
              >
                {t('auth.register')}
              </Link>
            </p>

            <p className="relative mt-6 flex items-center gap-1.5 text-xs text-text-disabled">
              <ShieldCheck className="size-3.5" />
              {t('app.subtitle')}
            </p>
          </div>
        </div>
      </main>
    </div>
  )
}
