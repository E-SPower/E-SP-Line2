import { useTranslation } from 'react-i18next'
import { ArrowUpRight, BookOpen, Code2, FileText, Layers } from 'lucide-react'
import { PageHeader } from '../components/ui/page-header'
import { Card, CardBody, CardHeader, CardTitle } from '../components/ui/card'

/** 文档都在仓库的 docs/ 下；用相对仓库根的路径，由 Vite 以静态文件托管。 */
const DOCS_BASE = '/docs'

const LINK_GROUPS = [
  {
    icon: BookOpen,
    titleKey: 'docs.userGuide',
    links: [
      { labelKey: 'docs.linkGettingStarted', href: `${DOCS_BASE}/user-guide/getting-started.md` },
      { labelKey: 'docs.linkAdapters', href: `${DOCS_BASE}/user-guide/adapters.md` },
      { labelKey: 'docs.linkRouting', href: `${DOCS_BASE}/user-guide/routing.md` },
      { labelKey: 'docs.linkTroubleshooting', href: `${DOCS_BASE}/user-guide/troubleshooting.md` },
    ],
  },
  {
    icon: Code2,
    titleKey: 'docs.apiDocs',
    links: [
      { labelKey: 'docs.linkApiReference', href: `${DOCS_BASE}/developer-guide/api-reference.md` },
      { labelKey: 'docs.linkProtocol', href: `${DOCS_BASE}/developer-guide/protocol-v3.md` },
      { labelKey: 'docs.linkMessageFields', href: `${DOCS_BASE}/developer-guide/message-fields.md` },
    ],
  },
  {
    icon: FileText,
    titleKey: 'docs.developerGuide',
    links: [
      { labelKey: 'docs.linkAdapterDevelopment', href: `${DOCS_BASE}/developer-guide/adapter-development.md` },
      { labelKey: 'docs.linkCapabilities', href: `${DOCS_BASE}/developer-guide/platform-capabilities.md` },
      { labelKey: 'docs.linkMigration', href: `${DOCS_BASE}/developer-guide/catbus-migration.md` },
    ],
  },
]

const LAYERS = [
  { titleKey: 'docs.layerPlatform', descKey: 'docs.layerPlatformDesc' },
  { titleKey: 'docs.layerAdapter', descKey: 'docs.layerAdapterDesc' },
  { titleKey: 'docs.layerInstance', descKey: 'docs.layerInstanceDesc' },
  { titleKey: 'docs.layerSession', descKey: 'docs.layerSessionDesc' },
]

export default function Docs() {
  const { t } = useTranslation()

  return (
    <>
      <PageHeader title={t('docs.title')} description={t('docs.subtitle')} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {LINK_GROUPS.map((group) => (
          <Card key={group.titleKey} padding="none">
            <CardHeader>
              <div className="flex items-center gap-2.5">
                <span aria-hidden className="grid size-7 place-items-center rounded-md border border-border bg-surface-inset text-text-secondary">
                  <group.icon className="size-4" />
                </span>
                <CardTitle className="text-base">{t(group.titleKey)}</CardTitle>
              </div>
            </CardHeader>
            <CardBody className="p-2">
              <ul>
                {group.links.map((link) => (
                  <li key={link.href}>
                    <a
                      href={link.href}
                      target="_blank"
                      rel="noreferrer"
                      className="group flex items-center justify-between gap-2 rounded-md px-2.5 py-2 text-sm text-text-secondary transition-colors hover:bg-surface-hover hover:text-text-primary"
                    >
                      <span className="truncate">{t(link.labelKey)}</span>
                      <ArrowUpRight className="size-3.5 shrink-0 text-text-disabled group-hover:text-text-tertiary" />
                    </a>
                  </li>
                ))}
              </ul>
            </CardBody>
          </Card>
        ))}
      </div>

      {/* Architecture — the one editorial, prose-like block. */}
      <Card padding="none">
        <CardHeader>
          <div className="flex items-center gap-2">
            <span aria-hidden className="grid size-7 place-items-center rounded-md bg-surface-inset text-text-secondary">
              <Layers className="size-4" />
            </span>
            <div>
              <CardTitle className="text-base">{t('docs.architecture')}</CardTitle>
              <p className="mt-0.5 text-xs text-text-tertiary">{t('docs.architectureHint')}</p>
            </div>
          </div>
        </CardHeader>
        <CardBody>
          <ol className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {LAYERS.map((layer, index) => (
              <li key={layer.titleKey} className="rounded-md border border-border-subtle bg-surface-sunken p-3">
                <div className="flex items-baseline gap-2">
                  <span className="tabular text-xs font-semibold text-text-disabled">
                    {String(index + 1).padStart(2, '0')}
                  </span>
                  <h3 className="text-sm font-medium text-text-primary">{t(layer.titleKey)}</h3>
                </div>
                <p className="mt-1 text-sm text-text-tertiary">{t(layer.descKey)}</p>
              </li>
            ))}
          </ol>
        </CardBody>
      </Card>
    </>
  )
}
