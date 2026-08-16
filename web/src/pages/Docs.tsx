import { useTranslation } from 'react-i18next'
import { useEffect, useMemo, useState } from 'react'
import { BookOpen, Code, FileText, Plug, Loader2, ChevronDown, ChevronRight } from 'lucide-react'
import apiClient from '../api/client'
import Markdown from '../components/Markdown'

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
  const [docs, setDocs] = useState<DocInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [openKeys, setOpenKeys] = useState<Set<string>>(new Set())

  // Load the index, then fetch each document's content so everything can be
  // shown on a single page (accordion), without navigating away.
  useEffect(() => {
    let cancelled = false
    apiClient.getDocs().then(async (res: any) => {
      if (cancelled) return
      if (res.error) {
        setError(res.error)
        setLoading(false)
        return
      }
      const index = Array.isArray(res.data) ? res.data : []
      const items: DocInfo[] = await Promise.all(
        index.map(async (d: any) => {
          const content = await apiClient.getDoc(d.key)
          return {
            key: d.key,
            title: d.title,
            group: d.group,
            content: content.data?.content || '',
          }
        })
      )
      if (cancelled) return
      setDocs(items)
      setLoading(false)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const toggle = (key: string) => {
    setOpenKeys((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const grouped = useMemo(
    () =>
      GROUPS.map((g) => ({
        ...g,
        items: docs.filter((d) => d.group === g.id),
      })).filter((g) => g.items.length > 0),
    [docs]
  )

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('docs.title')}</h1>
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
          <span className="ml-2 text-gray-500">{t('common.loading')}</span>
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
              <div key={group.id} className="bg-white rounded-lg shadow p-6">
                <div className="flex items-center mb-3">
                  <div className="p-3 bg-blue-100 rounded-lg">
                    <Icon className="w-6 h-6 text-blue-600" />
                  </div>
                  <div className="ml-3">
                    <h2 className="text-lg font-semibold text-gray-900">
                      {group.title}
                    </h2>
                    <p className="text-xs text-gray-500">{group.items.length} 篇文档</p>
                  </div>
                </div>

                <p className="text-sm text-gray-600 mb-4">{group.description}</p>

                <div className="space-y-1">
                  {group.items.map((doc) => {
                    const open = openKeys.has(doc.key)
                    return (
                      <div key={doc.key} className="border border-gray-100 rounded-lg overflow-hidden">
                        <button
                          onClick={() => toggle(doc.key)}
                          className="w-full flex items-center justify-between px-3 py-2 text-left text-sm hover:bg-gray-50"
                        >
                          <span className="text-blue-600 hover:text-blue-800">
                            {doc.title}
                          </span>
                          {open ? (
                            <ChevronDown className="w-4 h-4 text-gray-400 shrink-0" />
                          ) : (
                            <ChevronRight className="w-4 h-4 text-gray-400 shrink-0" />
                          )}
                        </button>
                        {open && (
                          <div className="px-4 py-3 border-t border-gray-100 max-h-96 overflow-y-auto">
                            <div className="prose max-w-none">
                              <Markdown source={doc.content} />
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
