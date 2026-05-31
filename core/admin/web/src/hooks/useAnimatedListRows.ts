import { useCallback, useEffect, useRef, useState } from 'react'

export type AnimatedListPhase = 'enter' | 'stable' | 'exit'

export type AnimatedListRow<T> = {
  key: string | number
  item: T
  phase: AnimatedListPhase
}

function motionMs(reduced: boolean, normal: number) {
  return reduced ? 0 : normal
}

export function useAnimatedListRows<T>(
  items: T[],
  itemKey: (item: T) => string | number,
) {
  const [rows, setRows] = useState<AnimatedListRow<T>[]>([])
  const mountedRef = useRef(false)
  const exitTimersRef = useRef<Map<string | number, ReturnType<typeof setTimeout>>>(new Map())
  const enterTimersRef = useRef<Map<string | number, ReturnType<typeof setTimeout>>>(new Map())
  const exitCallbacksRef = useRef<Map<string | number, () => void>>(new Map())
  const reducedMotionRef = useRef(
    typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  )

  const enterMs = motionMs(reducedMotionRef.current, 350)
  const exitMs = motionMs(reducedMotionRef.current, 300)

  const clearExitTimer = useCallback((key: string | number) => {
    const timer = exitTimersRef.current.get(key)
    if (timer) {
      clearTimeout(timer)
      exitTimersRef.current.delete(key)
    }
  }, [])

  const clearEnterTimer = useCallback((key: string | number) => {
    const timer = enterTimersRef.current.get(key)
    if (timer) {
      clearTimeout(timer)
      enterTimersRef.current.delete(key)
    }
  }, [])

  const scheduleEnterDone = useCallback(
    (key: string | number) => {
      clearEnterTimer(key)
      enterTimersRef.current.set(
        key,
        setTimeout(() => {
          enterTimersRef.current.delete(key)
          setRows((prev) =>
            prev.map((row) =>
              row.key === key && row.phase === 'enter' ? { ...row, phase: 'stable' } : row,
            ),
          )
        }, enterMs),
      )
    },
    [clearEnterTimer, enterMs],
  )

  const scheduleExitDone = useCallback(
    (key: string | number) => {
      clearExitTimer(key)
      exitTimersRef.current.set(
        key,
        setTimeout(() => {
          exitTimersRef.current.delete(key)
          const callback = exitCallbacksRef.current.get(key)
          exitCallbacksRef.current.delete(key)
          callback?.()
          setRows((prev) => prev.filter((row) => row.key !== key))
        }, exitMs),
      )
    },
    [clearExitTimer, exitMs],
  )

  const itemKeyRef = useRef(itemKey)
  itemKeyRef.current = itemKey

  useEffect(() => {
    const keyFn = itemKeyRef.current
    if (!mountedRef.current) {
      mountedRef.current = true
      setRows(items.map((item) => ({ key: keyFn(item), item, phase: 'stable' as const })))
      return
    }

    setRows((prev) => {
      const incomingSet = new Set(items.map((item) => keyFn(item)))
      const prevMap = new Map(prev.map((row) => [row.key, row]))
      const result: AnimatedListRow<T>[] = []

      for (const item of items) {
        const key = keyFn(item)
        const old = prevMap.get(key)
        if (old?.phase === 'exit') {
          result.push({ key, item, phase: 'enter' })
        } else if (old) {
          result.push({ key, item, phase: old.phase === 'enter' ? 'enter' : 'stable' })
        } else {
          result.push({ key, item, phase: 'enter' })
        }
      }

      for (const row of prev) {
        if (incomingSet.has(row.key)) continue
        if (row.phase === 'exit') {
          result.push(row)
        } else {
          result.push({ ...row, phase: 'exit' })
        }
      }

      return result
    })
  }, [items])

  useEffect(() => {
    for (const row of rows) {
      if (row.phase === 'enter' && !enterTimersRef.current.has(row.key)) {
        scheduleEnterDone(row.key)
      }
      if (row.phase === 'exit' && !exitTimersRef.current.has(row.key)) {
        scheduleExitDone(row.key)
      }
    }
  }, [rows, scheduleEnterDone, scheduleExitDone])

  useEffect(
    () => () => {
      for (const timer of exitTimersRef.current.values()) clearTimeout(timer)
      for (const timer of enterTimersRef.current.values()) clearTimeout(timer)
      exitTimersRef.current.clear()
      enterTimersRef.current.clear()
    },
    [],
  )

  const exitThen = useCallback((key: string | number, fn: () => void) => {
    if (exitCallbacksRef.current.has(key)) return
    exitCallbacksRef.current.set(key, fn)

    setRows((prev) => {
      const target = prev.find((row) => row.key === key)
      if (!target) {
        exitCallbacksRef.current.delete(key)
        fn()
        return prev
      }
      if (target.phase === 'exit') {
        return prev
      }
      return prev.map((row) => (row.key === key ? { ...row, phase: 'exit' as const } : row))
    })
  }, [])

  const flipKeys = rows.filter((row) => row.phase !== 'exit').map((row) => row.key)

  return { rows, flipKeys, exitThen }
}
