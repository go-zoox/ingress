import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { NavMenuGroup } from '../api/client'

export function useNavigation() {
  const [groups, setGroups] = useState<NavMenuGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    api
      .rbacMenus()
      .then((result) => {
        setGroups(result.groups ?? [])
        setLoading(false)
      })
      .catch((e: Error) => {
        setErr(e.message)
        setGroups([])
        setLoading(false)
      })
  }, [])

  useEffect(() => {
    load()
  }, [load])

  return { groups, loading, err, reload: load }
}
