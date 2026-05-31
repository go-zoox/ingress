import { OverviewTimeRangePicker } from './OverviewTimeRangePicker'
import { type OverviewRange, isRangeLiveEligible } from '../lib/overviewRange'

type Props = {
  metricsRange: OverviewRange
  onRangeChange: (range: OverviewRange) => void
  liveEnabled: boolean
  onLiveToggle: (enabled: boolean) => void
}

export function DetailMetricsToolbar({
  metricsRange,
  onRangeChange,
  liveEnabled,
  onLiveToggle,
}: Props) {
  const rangeLiveEligible = isRangeLiveEligible(metricsRange)

  return (
    <div className="overview-toolbar overview-toolbar-range">
      <div className="overview-toolbar-controls">
        <OverviewTimeRangePicker value={metricsRange} onChange={onRangeChange} />
        <label
          className="live-toggle overview-live-toggle"
          title={
            rangeLiveEligible
              ? '开启后增量刷新最新指标'
              : '所选历史区间已结束，无法实时更新'
          }
        >
          <input
            type="checkbox"
            checked={liveEnabled && rangeLiveEligible}
            disabled={!rangeLiveEligible}
            onChange={(e) => onLiveToggle(e.target.checked)}
          />
          <span>实时</span>
        </label>
      </div>
    </div>
  )
}
