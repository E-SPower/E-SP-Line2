import { useTranslation } from 'react-i18next'
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { BookOpen, Code, FileText, Plug, Loader2, ChevronRight, FileText as FileIcon } from 'lucide-react'
import apiClient from '../api/client'

interface DocInfo {
  key: string
  title: string
  group: string
  content: string
}

interface DocGroup {
  id: string
  title: string
  description: string
  icon: typeof BookOpen
}

const GROUPS: DocGroup[] = [
  {
    id: 'guide',
    title: '使用指南',
    description: '了解如何配置平台、接入器与实例，以及消息路由和常见问题',
    icon: BookOpen,
  },
  {
    id: 'adapters',
    title: '接入器文档',
    description: '闲鱼、淘宝等接入器的安装、配置、多开与 WebUI 启动控制',
    icon: Plug,
  },
  {
    id: 'api',
    title: 'API 参考',
    description: 'REST API 与 WebSocket 接口参考',
    icon: Code,
  },
  {
    id: 'dev',
    title: '开发者指南',
    description: '开发自定义接入器、协议规范与参与贡献',
    icon: FileText,
  },
]

export default function Docs() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [docs, setDocs] = useState<DocInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Load the index only (content is fetched lazily by DocViewer when the user
  // opens a document). This keeps the docs list fast.
  useEffect(() => {
    let cancelled = false
    apiClient.getDocs().then((res: any) => {
      if (cancelled) return
      if (res.error) {
        setError(res.error)
        setLoading(false)
        return
      }
      const index = Array.isArray(res.data) ? res.data : []
      setDocs(
        index.map((d: any) => ({
          key: d.key,
          title: d.title,
          group: d.group,
          content: '',
        }))
      )
      setLoading(false)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const grouped = useMemo(
    () =>
      GROUPS.map((g) => ({
        ...g,
        items: docs.filter((d) => d.group === g.id),
      })).filter((g) => g.items.length > 0),
    [docs]
  )

  const openDoc = (key: string) => {
    navigate(`/docs/${key}`)
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('docs.title')}</h1>
      </div>

      {/* 系统概述：简短说明这个框架是干什么的（置顶） */}
      <div className="mb-8 bg-gradient-to-r from-blue-600 to-blue-500 rounded-lg shadow p-6 text-white">
        <h2 className="text-lg font-semibold mb-2">E-SP-Line2 是什么</h2>
        <p className="text-sm leading-relaxed text-blue-50">
          一个平台化接入器管理系统：把闲鱼、淘宝等消息平台统一接入到一个后端，
          通过 Python 接入器接收各平台消息、转换为统一协议入库，再由路由规则分发到目标应用。
          多平台接入、多开多账号、启停控制全部在 WebUI 完成。
        </p>
      </div>

      {loading && (
        <div className="flex items-center justify-center h-40">
          <Loader2 className="w-6 h-6 text-blue-500 animate-spin" />
          <span className="ml-2 text-gray-500 dark:text-gray-400">{t('common.loading')}</span>
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <div className="text-red-800">Error: {error}</div>
        </div>
      )}

      {!loading && !error && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {grouped.map((group) => {
            const Icon = group.icon
            return (
              <div key={group.id} className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                <div className="flex items-center mb-3">
                  <div className="p-3 bg-blue-100 dark:bg-blue-900/40 rounded-lg">
                    <Icon className="w-6 h-6 text-blue-600 dark:text-blue-300" />
                  </div>
                  <div className="ml-3">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                      {group.title}
                    </h2>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{group.items.length} 篇文档</p>
                  </div>
                </div>

                <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">{group.description}</p>

                <div className="space-y-1">
                  {group.items.map((doc) => (
                    <button
                      key={doc.key}
                      onClick={() => openDoc(doc.key)}
                      className="w-full flex items-center justify-between px-3 py-2 text-left text-sm rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                    >
                      <span className="flex items-center text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300">
                        <FileIcon className="w-4 h-4 mr-2 shrink-0" />
                        {doc.title}
                      </span>
                      <ChevronRight className="w-4 h-4 text-gray-400 shrink-0" />
                    </button>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
