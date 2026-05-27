import { useCallback, useState } from 'react'
import { createPortal } from 'react-dom'

/** Single-line ellipsis cell with hover tooltip for full text. */
export function EllipsisTooltip({
  text,
  className = '',
  empty = '—',
}: {
  text: string
  className?: string
  empty?: string
}) {
  const [visible, setVisible] = useState(false)
  const [pos, setPos] = useState({ x: 0, y: 0 })

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

  const value = text.trim()
  if (!value) return <span className={className}>{empty}</span>

  const pop =
    visible &&
    createPortal(
      <div
        className="waf-rule-tooltip-pop waf-rule-tooltip-pop--fixed"
        role="tooltip"
        style={{ left: pos.x, top: pos.y }}
      >
        {value}
      </div>,
      document.body,
    )

  return (
    <>
      <span
        className={`ellipsis-tooltip ${className}`.trim()}
        onMouseEnter={onMouseEnter}
        onMouseLeave={hide}
        onFocus={onFocus}
        onBlur={hide}
        tabIndex={0}
      >
        {value}
      </span>
      {pop}
    </>
  )
}
