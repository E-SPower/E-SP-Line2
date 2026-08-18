import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Select from './Select'

export interface FieldConfig {
  key: string
  label: string
  type?: 'text' | 'password' | 'number' | 'select' | 'switch'
  required?: boolean
  placeholder?: string
  options?: { value: string; label: string }[]
  helpText?: string
  defaultValue?: any
  /** Custom renderer, overrides the default input rendering for this field. */
  render?: (value: any, setValue: (v: any) => void) => React.ReactNode
}

interface EntityModalProps {
  title: string
  open: boolean
  fields: FieldConfig[]
  initialValues?: Record<string, any>
  submitting?: boolean
  error?: string | null
  onClose: () => void
  onSubmit: (values: Record<string, any>) => void
  /** Notified whenever any field value changes. */
  onValuesChange?: (values: Record<string, any>) => void
}

// Shared control styling: neutral background, blue border on focus only.
const CONTROL_CLS =
  'w-full px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 ' +
  'rounded-lg text-gray-900 dark:text-gray-100 ' +
  'focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 ' +
  'placeholder-gray-400 dark:placeholder-gray-500'

// Reusable create/edit modal that renders a form from a field config.
export default function EntityModal({
  title,
  open,
  fields,
  initialValues,
  submitting,
  error,
  onClose,
  onSubmit,
  onValuesChange,
}: EntityModalProps) {
  const { t } = useTranslation()
  const [values, setValues] = useState<Record<string, any>>(
    () => (initialValues ? { ...initialValues } : {})
  )

  // Reset form values whenever the modal opens with new initial values.
  useEffect(() => {
    if (open) {
      const base = initialValues ? { ...initialValues } : {}
      fields.forEach((f) => {
        if (f.defaultValue !== undefined && base[f.key] === undefined) {
          base[f.key] = f.defaultValue
        }
      })
      setValues(base)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialValues])

  // When fields change dynamically (e.g. after choosing an adapter), apply
  // defaultValue only to fields that are not yet set. Do NOT reset values the
  // user has already filled in.
  useEffect(() => {
    if (!open) return
    setValues((prev) => {
      const next = { ...prev }
      let changed = false
      fields.forEach((f) => {
        if (f.defaultValue !== undefined && next[f.key] === undefined) {
          next[f.key] = f.defaultValue
          changed = true
        }
      })
      return changed ? next : prev
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fields, open])

  if (!open) return null

  const setField = (key: string, value: any) => {
    setValues((prev) => {
      const next = { ...prev, [key]: value }
      if (onValuesChange) {
        onValuesChange(next)
      }
      return next
    })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(values)
  }

  // Render a toggle switch for boolean fields.
  const renderSwitch = (field: FieldConfig) => {
    const checked = Boolean(values[field.key])
    return (
      <button
        type="button"
        onClick={() => setField(field.key, !checked)}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
          checked ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
        }`}
        role="switch"
        aria-checked={checked}
      >
        <span
          className={`inline-block h-5 w-5 transform rounded-full bg-white transition-transform ${
            checked ? 'translate-x-5' : 'translate-x-0.5'
          }`}
        />
      </button>
    )
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4">
        <div
          className="fixed inset-0 bg-black bg-opacity-50"
          onClick={onClose}
        />
        <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg">
          <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
            <button
              type="button"
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
            >
              ✕
            </button>
          </div>

          <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
            {fields.map((field) => (
              <div key={field.key}>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {field.label}
                  {field.required && <span className="text-red-500"> *</span>}
                </label>
                {field.render ? (
                  field.render(values[field.key], (v) => setField(field.key, v))
                ) : field.type === 'switch' ? (
                  <div className="flex items-center space-x-3">
                    {renderSwitch(field)}
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      {values[field.key] ? t('common.yes') : t('common.no')}
                    </span>
                  </div>
                ) : field.type === 'select' ? (
                  <Select
                    value={values[field.key] ?? ''}
                    onChange={(v) => setField(field.key, v)}
                    options={field.options || []}
                    placeholder={field.placeholder}
                  />
                ) : (
                  <input
                    type={
                      field.type === 'number'
                        ? 'number'
                        : field.type === 'password'
                        ? 'password'
                        : 'text'
                    }
                    value={values[field.key] ?? ''}
                    onChange={(e) =>
                      setField(
                        field.key,
                        field.type === 'number'
                          ? Number(e.target.value)
                          : e.target.value
                      )
                    }
                    placeholder={field.placeholder}
                    required={field.required}
                    className={CONTROL_CLS}
                  />
                )}
                {field.helpText && (
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{field.helpText}</p>
                )}
              </div>
            ))}

            {error && (
              <div className="rounded-md bg-red-50 dark:bg-red-900/30 p-3">
                <p className="text-sm text-red-800 dark:text-red-300">{error}</p>
              </div>
            )}

            <div className="flex justify-end space-x-3 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {submitting ? t('common.saving') : t('common.save')}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
