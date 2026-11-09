import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

export type Theme = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

/** Must stay in sync with the inline bootstrap script in index.html. */
const DEFAULT_STORAGE_KEY = 'espl.theme'

interface ThemeContextValue {
  /** What the user picked — may be `system`. */
  theme: Theme
  /** What is actually painted right now — never `system`. */
  resolvedTheme: ResolvedTheme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function readStoredTheme(storageKey: string, fallback: Theme): Theme {
  if (typeof window === 'undefined') return fallback
  try {
    const stored = window.localStorage.getItem(storageKey)
    return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : fallback
  } catch {
    return fallback
  }
}

function systemPrefersDark(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function resolve(theme: Theme): ResolvedTheme {
  if (theme === 'system') return systemPrefersDark() ? 'dark' : 'light'
  return theme
}

function applyClass(resolved: ResolvedTheme) {
  const root = document.documentElement
  root.classList.toggle('dark', resolved === 'dark')
  // Keep the mobile browser chrome in step with the page.
  root.style.colorScheme = resolved
  // Same for `<meta name="theme-color">`, which the bootstrap script also sets.
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  meta?.setAttribute('content', resolved === 'dark' ? '#0a0a0a' : '#ffffff')
}

export interface ThemeProviderProps {
  children: ReactNode
  /** Defaults to `system`; a stored preference wins when present. */
  defaultTheme?: Theme
  /** localStorage key override, mainly for tests. */
  storageKey?: string
}

/**
 * Owns the light/dark preference: reads localStorage + `prefers-color-scheme`,
 * writes `.dark` and `color-scheme` on `<html>`, and follows the OS while the
 * preference is `system`. Pages only ever call `useTheme()`.
 */
export function ThemeProvider({
  children,
  defaultTheme = 'system',
  storageKey = DEFAULT_STORAGE_KEY,
}: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(() => readStoredTheme(storageKey, defaultTheme))
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() =>
    resolve(readStoredTheme(storageKey, defaultTheme)),
  )

  // Paint before the first browser frame so there is no light flash.
  useEffect(() => {
    const next = resolve(theme)
    setResolvedTheme(next)
    applyClass(next)
  }, [theme])

  // Follow the OS while the preference is `system`.
  useEffect(() => {
    if (theme !== 'system' || !window.matchMedia) return
    const query = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => {
      const next = resolve('system')
      setResolvedTheme(next)
      applyClass(next)
    }
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  }, [theme])

  // Two tabs, one preference.
  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== storageKey) return
      const next: Theme =
        event.newValue === 'light' || event.newValue === 'dark' || event.newValue === 'system'
          ? event.newValue
          : 'system'
      setThemeState(next)
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [storageKey])

  const setTheme = useCallback(
    (next: Theme) => {
      try {
        window.localStorage.setItem(storageKey, next)
      } catch {
        // Private mode / quota: still apply it for this session.
      }
      setThemeState(next)
    },
    [storageKey],
  )

  const toggleTheme = useCallback(() => {
    setTheme(resolve(theme) === 'dark' ? 'light' : 'dark')
  }, [setTheme, theme])

  const value = useMemo<ThemeContextValue>(
    () => ({ theme, resolvedTheme, setTheme, toggleTheme }),
    [theme, resolvedTheme, setTheme, toggleTheme],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used inside <ThemeProvider>')
  }
  return context
}

export default ThemeProvider
