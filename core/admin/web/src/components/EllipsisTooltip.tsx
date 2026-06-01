import { useCallback, useState, type FocusEvent, type MouseEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

function useFixedTooltip() {
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
    (e: MouseEvent<HTMLElement>) => showAt(e.currentTarget),
    [showAt],
  )
  const onFocus = useCallback(
    (e: FocusEvent<HTMLElement>) => showAt(e.currentTarget),
    [showAt],
  )
  const hide = useCallback(() => setVisible(false), [])

  return { visible, pos, onMouseEnter, onFocus, hide }
}

function FixedTooltipPop({
  visible,
  pos,
  children,
}: {
  visible: boolean
  pos: { x: number; y: number }
  children: ReactNode
}) {
  if (!visible) return null
  return createPortal(
    <div
      className="waf-rule-tooltip-pop waf-rule-tooltip-pop--fixed"
      role="tooltip"
      style={{ left: pos.x, top: pos.y }}
    >
      {children}
    </div>,
    document.body,
  )
}

/** Wraps children; shows `content` in a tooltip on hover/focus (no inline text). */
export function HoverTooltip({
  content,
  children,
  className = '',
}: {
  content: string
  children: ReactNode
  className?: string
}) {
  const value = content.trim()
  const { visible, pos, onMouseEnter, onFocus, hide } = useFixedTooltip()

  if (!value) {
    return <>{children}</>
  }

  return (
    <>
      <span
        className={`hover-tooltip-trigger ${className}`.trim()}
        onMouseEnter={onMouseEnter}
        onMouseLeave={hide}
        onFocus={onFocus}
        onBlur={hide}
        tabIndex={0}
      >
        {children}
      </span>
      <FixedTooltipPop visible={visible} pos={pos}>
        {value}
      </FixedTooltipPop>
    </>
  )
}

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
  const { visible, pos, onMouseEnter, onFocus, hide } = useFixedTooltip()

  const value = text.trim()
  if (!value) return <span className={className}>{empty}</span>

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
      <FixedTooltipPop visible={visible} pos={pos}>
        {value}
      </FixedTooltipPop>
    </>
  )
}
