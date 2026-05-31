import { useCallback, useLayoutEffect, useRef } from 'react'

const REORDER_MS = 350

function prefersReducedMotion() {
  return (
    typeof window !== 'undefined' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

/** FLIP layout animation when list item order changes (not external layout shifts). */
export function useListFlip(keys: (string | number)[]) {
  const positionsRef = useRef(new Map<string | number, { top: number; left: number }>())
  const nodesRef = useRef(new Map<string | number, HTMLElement>())
  const prevOrderRef = useRef<(string | number)[]>([])
  const keysToken = keys.join('\0')

  const register = useCallback(
    (key: string | number) => (el: HTMLElement | null) => {
      if (el) nodesRef.current.set(key, el)
      else nodesRef.current.delete(key)
    },
    [],
  )

  useLayoutEffect(() => {
    const nextPos = new Map<string | number, { top: number; left: number }>()
    for (const key of keys) {
      const el = nodesRef.current.get(key)
      if (!el) continue
      const rect = el.getBoundingClientRect()
      nextPos.set(key, { top: rect.top, left: rect.left })
    }

    const prevOrder = prevOrderRef.current
    const orderChanged =
      keys.length > 0 &&
      (keys.length !== prevOrder.length || keys.some((key, i) => key !== prevOrder[i]))
    prevOrderRef.current = keys.slice()

    if (!orderChanged || prefersReducedMotion()) {
      positionsRef.current = nextPos
      return
    }

    for (const key of keys) {
      const el = nodesRef.current.get(key)
      const prev = positionsRef.current.get(key)
      const next = nextPos.get(key)
      if (!el || !prev || !next) continue

      const dx = prev.left - next.left
      const dy = prev.top - next.top
      if (Math.abs(dx) < 0.5 && Math.abs(dy) < 0.5) continue

      el.style.transform = `translate(${dx}px, ${dy}px)`
      el.style.transition = 'transform 0s'
      void el.getBoundingClientRect()
      requestAnimationFrame(() => {
        el.style.transition = `transform ${REORDER_MS}ms cubic-bezier(0.22, 1, 0.36, 1)`
        el.style.transform = ''
      })
    }

    positionsRef.current = nextPos
  }, [keysToken])

  return register
}
