import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import apiClient from '../api/client'

export interface FormOption {
  value: string
  label: string
  label_en?: string
}

export interface FormOptionGroup {
  key: string
  options: FormOption[]
}

/**
 * Loads all form option groups from the backend YAML registry and provides
 * helper methods to build select options with i18n-aware labels.
 */
export function useFormOptions() {
  const { i18n } = useTranslation()
  const [groups, setGroups] = useState<FormOptionGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    const result = await apiClient.getFormOptions()
    if (result.error) {
      setError(result.error)
      setGroups([])
    } else {
      setGroups(result.data || [])
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const isZh = i18n.language?.toLowerCase().startsWith('zh')

  const getGroup = useCallback(
    (key: string): FormOptionGroup | undefined => groups.find((g) => g.key === key),
    [groups]
  )

  const getOptions = useCallback(
    (key: string): { value: string; label: string }[] => {
      const group = getGroup(key)
      if (!group) return []
      return group.options.map((opt) => ({
        value: opt.value,
        label: isZh ? opt.label : opt.label_en || opt.label,
      }))
    },
    [getGroup, isZh]
  )

  return { groups, loading, error, reload: load, getGroup, getOptions }
}
