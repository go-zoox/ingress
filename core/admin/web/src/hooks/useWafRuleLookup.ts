import { useEffect, useMemo, useState } from 'react'
import { api, type WAFRuleDetail } from '../api/client'
import { buildWafRuleLookup } from '../lib/wafRuleTooltip'

export function useWafRuleLookup() {
  const [catalog, setCatalog] = useState<WAFRuleDetail[]>([])

  useEffect(() => {
    api.wafRulesCatalog().then((rows) => setCatalog(Array.isArray(rows) ? rows : [])).catch(() => setCatalog([]))
  }, [])

  const lookup = useMemo(() => buildWafRuleLookup(catalog), [catalog])

  return { catalog, lookup }
}
