import { useTranslation } from 'react-i18next'
import { BookOpen, Code, FileText } from 'lucide-react'

export default function Docs() {
  const { t } = useTranslation()

  const docSections = [
    {
      icon: BookOpen,
      title: t('docs.userGuide'),
      description: '了解如何使用 E-SP-Line2 平台管理您的接入器和实例',
      links: [
        { label: '快速开始', href: '#' },
        { label: '平台配置', href: '#' },
        { label: '接入器管理', href: '#' },
        { label: '消息路由', href: '#' }
      ]
    },
    {
      icon: Code,
      title: t('docs.apiDocs'),
      description: '完整的 API 文档，包括 REST API 和 WebSocket 接口',
      links: [
        { label: 'REST API', href: '#' },
        { label: 'WebSocket API', href: '#' },
        { label: '认证授权', href: '#' },
        { label: '错误码说明', href: '#' }
      ]
    },
    {
      icon: FileText,
      title: t('docs.developerGuide'),
      description: '开发者指南，帮助您开发和集成自定义接入器',
      links: [
        { label: '接入器开发', href: '#' },
        { label: '协议规范', href: '#' },
        { label: '最佳实践', href: '#' },
        { label: '示例代码', href: '#' }
      ]
    }
  ]

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('docs.title')}</h1>
        <p className="mt-2 text-gray-600">
          欢迎使用 E-SP-Line2 文档中心，这里提供了完整的使用指南、API 文档和开发者指南
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {docSections.map((section, index) => {
          const Icon = section.icon
          return (
            <div key={index} className="bg-white rounded-lg shadow p-6">
              <div className="flex items-center mb-4">
                <div className="p-3 bg-blue-100 rounded-lg">
                  <Icon className="w-6 h-6 text-blue-600" />
                </div>
                <h2 className="ml-3 text-lg font-semibold text-gray-900">
                  {section.title}
                </h2>
              </div>
              
              <p className="text-sm text-gray-600 mb-4">
                {section.description}
              </p>

              <ul className="space-y-2">
                {section.links.map((link, linkIndex) => (
                  <li key={linkIndex}>
                    <a
                      href={link.href}
                      className="text-sm text-blue-600 hover:text-blue-800 hover:underline"
                    >
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          )
        })}
      </div>

      <div className="mt-8 bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          系统架构说明
        </h2>
        <div className="prose max-w-none">
          <p className="text-gray-600">
            E-SP-Line2 是一个平台化接入器管理系统，采用四层架构设计：
          </p>
          <ul className="mt-4 space-y-2 text-gray-600">
            <li><strong>平台层</strong>：定义业务平台类型（闲鱼、淘宝、千牛等）</li>
            <li><strong>接入器包层</strong>：具体平台的接入实现，带版本和能力声明</li>
            <li><strong>接入器实例层</strong>：真实账号或店铺的运行配置</li>
            <li><strong>运行会话层</strong>：实例当前在线连接状态</li>
          </ul>
          <p className="mt-4 text-gray-600">
            系统支持异步消息处理、多重可用性保障、多语言国际化等特性。
          </p>
        </div>
      </div>
    </div>
  )
}
