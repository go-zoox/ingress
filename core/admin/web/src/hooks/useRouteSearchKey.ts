import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

/** Serialized query string; use as effect dep when syncing local state to the URL. */
export function useRouteSearchKey(): string {
  const [searchParams] = useSearchParams()
  return useMemo(() => searchParams.toString(), [searchParams])
}
