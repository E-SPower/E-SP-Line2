import { useTranslation } from 'react-i18next'
import { ArrowUpRight, BookOpen, Code2, FileText, Layers } from 'lucide-react'
import { PageHeader } from '../components/ui/page-header'
import { Card, CardBody, CardHeader, CardTitle } from '../components/ui/card'
import { EmptyState } from '../components/ui/empty-state'

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

      {/* `minmax(0, 1fr)` tracks: an unbreakable group title must not widen
          its column past a third of the rail on a narrow window. */}
      <div className="grid min-w-0 grid-cols-1 gap-4 lg:grid-cols-[repeat(3,minmax(0,1fr))]">
        {LINK_GROUPS.map((group) => (
          <Card key={group.titleKey} padding="none">
            <CardHeader>
              <div className="flex min-w-0 items-center gap-2.5">
                <span aria-hidden className="grid size-7 shrink-0 place-items-center rounded-md border border-border bg-surface-inset text-text-secondary">
                  <group.icon className="size-4" />
                </span>
                <CardTitle className="min-w-0 break-words text-base">{t(group.titleKey)}</CardTitle>
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
          <div className="flex min-w-0 items-center gap-2">
            <span aria-hidden className="grid size-7 shrink-0 place-items-center rounded-md bg-surface-inset text-text-secondary">
              <Layers className="size-4" />
            </span>
            <div className="min-w-0">
              <CardTitle className="break-words text-base">{t('docs.architecture')}</CardTitle>
              <p className="mt-0.5 break-words text-xs text-text-tertiary">{t('docs.architectureHint')}</p>
            </div>
          </div>
        </CardHeader>
        <CardBody>
          <ol className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {LAYERS.map((layer, index) => (
              <li key={layer.titleKey} className="min-w-0 rounded-md border border-border-subtle bg-surface-sunken p-3">
                <div className="flex min-w-0 items-baseline gap-2">
                  <span className="tabular shrink-0 text-xs font-semibold text-text-disabled">
                    {String(index + 1).padStart(2, '0')}
                  </span>
                  <h3 className="min-w-0 break-words text-sm font-medium text-text-primary">{t(layer.titleKey)}</h3>
                </div>
                <p className="mt-1 break-words text-sm text-text-tertiary">{t(layer.descKey)}</p>
              </li>
            ))}
          </ol>
        </CardBody>
      </Card>

      {/* The page used to bottom out in a blank half-screen; one page-sized
          blank carries the faint aurora instead, reusing the login primitives. */}
      <EmptyState
        icon={BookOpen}
        size="page"
        aurora
        title={t('docs.more')}
        description={t('docs.moreHint')}
      />
    </>
  )
}
