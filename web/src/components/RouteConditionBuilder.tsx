import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import Select from './Select'

export interface ConditionRow {
  field: string
  value: string
}

interface RouteConditionBuilderProps {
  value: ConditionRow[]
  onChange: (rows: ConditionRow[]) => void
  /** Adapter options for the adapter_id condition, loaded from the API. */
  adapterOptions: { value: string; label: string }[]
}

// Event types supported by the routing engine (see internal/protocol/v3/envelope.go)
const EVENT_TYPES = [
  'message.received',
  'message.sent',
  'message.delivered',
  'message.acked',
  'message.failed',
  'adapter.connected',
  'adapter.disconnected',
  'adapter.started',
  'adapter.stopped',
  'adapter.error',
  'command.created',
  'command.sent',
  'command.failed',
  'system.error',
  'system.health_check',
]

// Convert the backend conditions map ({ event_type: "...", adapter_id: "..." })
// into visual rows. The backend router only supports equality matching.
export function conditionsToRows(conditions: string | null | undefined): ConditionRow[] {
  if (!conditions) return []
  try {
    const parsed = JSON.parse(conditions)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const rows: ConditionRow[] = []
      if (typeof parsed.event_type === 'string' && parsed.event_type) {
        rows.push({ field: 'event_type', value: parsed.event_type })
      }
      if (typeof parsed.adapter_id === 'string' && parsed.adapter_id) {
        rows.push({ field: 'adapter_id', value: parsed.adapter_id })
      }
      return rows
    }
  } catch {
    // Ignore malformed JSON
  }
  return []
}

// Convert visual rows back into the backend conditions map.
export function rowsToConditions(rows: ConditionRow[]): string {
  const map: Record<string, string> = {}
  rows.forEach((row) => {
    if (row.field && row.value) {
      map[row.field] = row.value
    }
  })
  return JSON.stringify(map)
}

/**
 * Visual condition builder for route rules. Each row is a single equality
 * condition (e.g. "事件类型 = 收到消息"). Rows are serialized to the backend
 * conditions JSON map, so users never hand-write JSON. Event types are shown
 * with localized labels and descriptions via i18n.
 */
export default function RouteConditionBuilder({
  value,
  onChange,
  adapterOptions,
}: RouteConditionBuilderProps) {
  const { t } = useTranslation()

  const eventTypeLabel = (et: string) => {
    const key = `routes.eventTypes.${et}`
    const label = t(key)
    // If the key resolves to itself, no translation exists -> show raw code.
    return label === key ? et : label
  }

  const eventTypeDesc = (et: string) => {
    const key = `routes.eventTypes.${et}.desc`
    const desc = t(key)
    return desc === key ? '' : desc
  }

  const updateRow = (index: number, patch: Partial<ConditionRow>) => {
    const next = value.map((row, i) => (i === index ? { ...row, ...patch } : row))
    onChange(next)
  }

  const addRow = () => {
    onChange([...value, { field: 'event_type', value: EVENT_TYPES[0] }])
  }

  const removeRow = (index: number) => {
    onChange(value.filter((_, i) => i !== index))
  }

  const fieldOptions = [
    { value: 'event_type', label: t('routes.eventTypesField') },
    { value: 'adapter_id', label: t('routes.adapterField') },
  ]

  return (
    <div className="space-y-2">
      {value.length === 0 && (
        <p className="text-sm text-gray-500">{t('routes.noConditionsHint')}</p>
      )}

      {value.map((row, index) => (
        <div key={index}>
          <div className="flex items-center space-x-2">
            <Select
              value={row.field}
              onChange={(v) => updateRow(index, { field: v })}
              options={fieldOptions}
              className="w-40"
            />

            <span className="text-sm text-gray-500 dark:text-gray-400">{t('routes.equals')}</span>

            {row.field === 'event_type' ? (
              <Select
                value={row.value}
                onChange={(v) => updateRow(index, { value: v })}
                options={EVENT_TYPES.map((et) => ({ value: et, label: eventTypeLabel(et) }))}
                className="flex-1"
              />
            ) : (
              <Select
                value={row.value}
                onChange={(v) => updateRow(index, { value: v })}
                options={adapterOptions}
                placeholder={t('routes.selectAdapter')}
                className="flex-1"
              />
            )}

            <button
              type="button"
              onClick={() => removeRow(index)}
              className="p-1.5 text-red-500 hover:bg-red-50 rounded"
              aria-label={t('routes.removeCondition')}
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>

          {row.field === 'event_type' && eventTypeDesc(row.value) && (
            <p className="mt-1 ml-1 text-xs text-gray-500">
              {eventTypeDesc(row.value)}
            </p>
          )}
        </div>
      ))}

      <button
        type="button"
        onClick={addRow}
        className="flex items-center px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 rounded-lg border border-blue-200"
      >
        <Plus className="w-4 h-4 mr-1" />
        {t('routes.addCondition')}
      </button>

      <p className="text-xs text-gray-500">{t('routes.conditionsAndHint')}</p>
    </div>
  )
}
