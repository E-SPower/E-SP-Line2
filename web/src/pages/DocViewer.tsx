import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, BookOpen } from 'lucide-react'
import apiClient from '../api/client'
import Markdown from '../components/Markdown'

interface DocInfo {
  key: string
  title: string
}

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
        setDocs(res.data)
      }
    })
  }, [])

  useEffect(() => {
    if (!key) return
    setLoading(true)
    setError(null)
    apiClient.getDoc(key).then((res: any) => {
      setLoading(false)
      if (res.error) {
        setError(res.error)
        return
      }
      setDoc(res.data)
    })
  }, [key])

  const currentKey = doc?.key || key

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            {doc?.title || t('docs.title')}
          </h1>
          <p className="mt-2 text-gray-600">{doc?.path}</p>
        </div>
        <button
          onClick={() => navigate('/docs')}
          className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-lg text-sm text-gray-700 hover:bg-gray-50"
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
          <div className="text-gray-500">{t('common.loading')}</div>
        </div>
      )}

      {!loading && !error && doc && (
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Table of contents sidebar */}
          <aside className="hidden lg:block">
            <div className="bg-white rounded-lg shadow p-4 sticky top-4">
              <h3 className="text-sm font-semibold text-gray-700 mb-3 flex items-center">
                <BookOpen className="w-4 h-4 mr-2 text-blue-600" />
                {t('docs.toc') || '目录'}
              </h3>
              <ul className="space-y-1 text-sm">
                {docs.map((d) => (
                  <li key={d.key}>
                    <a
                      href={`/docs/${d.key}`}
                      onClick={(e) => {
                        e.preventDefault()
                        navigate(`/docs/${d.key}`)
                      }}
                      className={`block px-2 py-1.5 rounded hover:bg-gray-50 ${
                        d.key === currentKey
                          ? 'text-blue-600 font-medium bg-blue-50'
                          : 'text-gray-600'
                      }`}
                    >
                      {d.title}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          </aside>

          {/* Document content */}
          <article className="lg:col-span-3">
            <div className="bg-white rounded-lg shadow p-6 md:p-8">
              <Markdown source={doc.content} />
            </div>
          </article>
        </div>
      )}
    </div>
  )
}
