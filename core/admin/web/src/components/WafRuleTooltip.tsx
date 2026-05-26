import { useCallback, useState } from 'react'
import { createPortal } from 'react-dom'
import type { WAFRuleDetail } from '../api/client'
import { formatWafRuleTooltip, resolveWafRule } from '../lib/wafRuleTooltip'

type Props = {
  rule: string
  lookup: Map<string, WAFRuleDetail>
  className?: string
}

/** Hover tooltip (fixed layer) showing WAF rule definition for a rule id/phase string. */
export function WafRuleTooltip({ rule, lookup, className = '' }: Props) {
  const [visible, setVisible] = useState(false)
  const [pos, setPos] = useState({ x: 0, y: 0 })

  const detail = resolveWafRule(lookup, rule)
  const tip = formatWafRuleTooltip(detail, rule)

  const showAt = useCallback((el: HTMLElement) => {
    const rect = el.getBoundingClientRect()
    const popW = 360
    let x = rect.left
    const y = rect.bottom + 8
    if (x + popW > window.innerWidth - 12) {
      x = Math.max(12, window.innerWidth - popW - 12)
    }
    setPos({ x, y })
    setVisible(true)
  }, [])

  const onMouseEnter = useCallback(
    (e: React.MouseEvent<HTMLSpanElement>) => showAt(e.currentTarget),
    [showAt],
  )
  const onFocus = useCallback(
    (e: React.FocusEvent<HTMLSpanElement>) => showAt(e.currentTarget),
    [showAt],
  )

  const hide = useCallback(() => setVisible(false), [])

  if (!rule) return <span className={className}>—</span>

  const pop =
    visible &&
    createPortal(
      <div
        className="waf-rule-tooltip-pop waf-rule-tooltip-pop--fixed"
        role="tooltip"
        style={{ left: pos.x, top: pos.y }}
      >
        {tip.split('\n').map((line, i) => (
          <span key={i} className="waf-rule-tooltip-line">
            {line}
          </span>
        ))}
      </div>,
      document.body,
    )

  return (
    <>
      <span
        className={`waf-rule-tooltip ${className}`.trim()}
        onMouseEnter={onMouseEnter}
        onMouseLeave={hide}
        onFocus={onFocus}
        onBlur={hide}
        onClick={(e) => e.stopPropagation()}
        tabIndex={0}
      >
        <span className="waf-rule-tooltip-label">{rule}</span>
      </span>
      {pop}
    </>
  )
}
