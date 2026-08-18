import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, BookOpen } from 'lucide-react'
import apiClient from '../api/client'
import Markdown from '../components/Markdown'

interface DocInfo {
  key: string
  title: string
  group: string
  path?: string
}

// Group display labels for the sidebar (kept in sync with the Docs page).
const GROUP_LABELS: Record<string, string> = {
  guide: '使用指南',
  adapters: '接入器文档',
  api: 'API 参考',
  dev: '开发者指南',
}

// Simple in-memory cache for document contents to avoid re-fetching on every
// navigation between docs.
const contentCache = new Map<string, any>()

export default function DocViewer() {
  const { t } = useTranslation()
  const { key } = useParams<{ key: string }>()
  const navigate = useNavigate()

  const [doc, setDoc] = useState<any>(null)
  const [docs, setDocs] = useState<DocInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    apiClient.getDocs().then((res: any) => {
      if (!res.error && Array.isArray(res.data)) {
        setDocs(
          res.data.map((d: any) => ({ key: d.key, title: d.title, group: d.group, path: d.path }))
        )
      }
    })
  }, [])

  useEffect(() => {
    if (!key) return
    setLoading(true)
    setError(null)

    // Serve from cache when possible.
    if (contentCache.has(key)) {
      setDoc(contentCache.get(key))
      setLoading(false)
      return
    }

    apiClient.getDoc(key).then((res: any) => {
      setLoading(false)
      if (res.error) {
        setError(res.error)
        return
      }
      contentCache.set(key, res.data)
      setDoc(res.data)
    })
  }, [key])

  const currentKey = doc?.key || key

  // Group docs for the sidebar, preserving order.
  const groups = ['guide', 'adapters', 'api', 'dev']
    .map((g) => ({
      id: g,
      label: GROUP_LABELS[g] || g,
      items: docs.filter((d) => d.group === g),
    }))
    .filter((g) => g.items.length > 0)

  // Resolve a markdown link target to a doc key. Matches by:
  //   1. doc key (e.g. "adapter-yaml", "/docs/protocol")
  //   2. doc path filename (e.g. "./adapters.md" -> user-guide/adapters.md)
  const resolveDocKey = (target: string): string | null => {
    const clean = target.replace(/\.md$/, '').replace(/^\.?\//, '').replace(/\/$/, '')
    // Direct key match.
    if (docs.some((d) => d.key === clean)) return clean
    // Match by path basename (file name without directory).
    const base = clean.split('/').pop() || clean
    const byPath = docs.find((d) => {
      const p = (d as any).path || ''
      return p.replace(/\.md$/, '') === clean || p.endsWith('/' + base + '.md')
    })
    return byPath ? byPath.key : null
  }

  // Handle internal markdown links: navigate via the router for /docs/<key>
  // links, bare doc keys and relative "./xxx.md" references; open http(s)
  // links in a new tab.
  const handleLinkClick = (href: string) => {
    // Normalize: strip surrounding slashes and "./" prefixes.
    let target = href.trim()
    if (target.startsWith('/')) target = target.slice(1)
    if (target.startsWith('docs/')) {
      const docKey = target.slice('docs/'.length)
      const resolved = resolveDocKey(docKey)
      if (resolved) {
        navigate(`/docs/${resolved}`)
      }
      return
    }
    const resolved = resolveDocKey(target)
    if (resolved) {
      navigate(`/docs/${resolved}`)
      return
    }
    // Otherwise open in a new tab (external / unknown).
    window.open(href, '_blank', 'noopener,noreferrer')
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {doc?.title || t('docs.title')}
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">{doc?.path}</p>
        </div>
        <button
          onClick={() => navigate('/docs')}
          className="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
        >
          <ArrowLeft className="w-4 h-4 mr-2" />
          {t('docs.backToList') || 'Back to docs'}
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <div className="text-red-800">Error: {error}</div>
        </div>
      )}

      {loading && (
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-500 dark:text-gray-400">{t('common.loading')}</div>
        </div>
      )}

      {!loading && !error && doc && (
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Table of contents sidebar (grouped) */}
          <aside className="hidden lg:block">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 sticky top-4">
              <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3 flex items-center">
                <BookOpen className="w-4 h-4 mr-2 text-blue-600 dark:text-blue-400" />
                {t('docs.toc') || '目录'}
              </h3>
              {groups.map((group) => (
                <div key={group.id} className="mb-4">
                  <p className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">
                    {group.label}
                  </p>
                  <ul className="space-y-1 text-sm">
                    {group.items.map((d) => (
                      <li key={d.key}>
                        <a
                          href={`/docs/${d.key}`}
                          onClick={(e) => {
                            e.preventDefault()
                            navigate(`/docs/${d.key}`)
                          }}
                          className={`block px-2 py-1.5 rounded hover:bg-gray-50 dark:hover:bg-gray-700 ${
                            d.key === currentKey
                              ? 'text-blue-600 dark:text-blue-400 font-medium bg-blue-50 dark:bg-blue-900/30'
                              : 'text-gray-600 dark:text-gray-300'
                          }`}
                        >
                          {d.title}
                        </a>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </aside>

          {/* Document content */}
          <article className="lg:col-span-3">
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 md:p-8">
              <Markdown source={doc.content} onLinkClick={handleLinkClick} />
            </div>
          </article>
        </div>
      )}
    </div>
  )
}
