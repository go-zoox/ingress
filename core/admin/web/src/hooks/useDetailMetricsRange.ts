import { useCallback, useMemo, useState } from 'react'
import { loadPreferences, savePreferences } from '../lib/preferences'
import {
  type OverviewRange,
  isRangeLiveEligible,
  parseOverviewRange,
  rangeToQueryParams,
  serializeOverviewRange,
} from '../lib/overviewRange'

export function useDetailMetricsRange(options?: { defaultLive?: boolean }) {
  const prefs = loadPreferences()
  const defaultLive = options?.defaultLive ?? false

  const [metricsRange, setMetricsRangeState] = useState<OverviewRange>(() =>
    parseOverviewRange(prefs.overviewRange, prefs.metricsWindow),
  )
  const [liveEnabled, setLiveEnabledState] = useState(() => {
    if (defaultLive) {
      return prefs.overviewLiveEnabled !== false
    }
    return prefs.detailLiveEnabled ?? false
  })

  const rangeLiveEligible = isRangeLiveEligible(metricsRange)
  const streaming = liveEnabled && rangeLiveEligible
  const rangeQuery = useMemo(() => rangeToQueryParams(metricsRange), [metricsRange])

  const setMetricsRange = useCallback((next: OverviewRange) => {
    setMetricsRangeState(next)
    const p = loadPreferences()
    savePreferences({
      ...p,
      metricsWindow: next.kind === 'preset' ? next.preset : 'custom',
      overviewRange: serializeOverviewRange(next),
    })
  }, [])

  const setLiveEnabled = useCallback(
    (next: boolean) => {
      setLiveEnabledState(next)
      const p = loadPreferences()
      if (defaultLive) {
        savePreferences({ ...p, overviewLiveEnabled: next })
      } else {
        savePreferences({ ...p, detailLiveEnabled: next })
      }
    },
    [defaultLive],
  )

  return {
    metricsRange,
    setMetricsRange,
    liveEnabled,
    setLiveEnabled,
    streaming,
    rangeQuery,
  }
}
