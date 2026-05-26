import { Navigate, useLocation } from 'react-router-dom'

/** Legacy /topology URL → routes topology tab */
export function TopologyPage() {
  const { search } = useLocation()
  const extra = search ? search.replace(/^\?/, '&') : ''
  return <Navigate to={`/routes?tab=topology${extra}`} replace />
}
