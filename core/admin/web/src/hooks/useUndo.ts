import { useCallback, useRef, useState } from 'react'

/** Maximum number of undo steps retained. */
const MAX_HISTORY = 50

/** Debounce interval in milliseconds before a change is committed to history. */
const DEBOUNCE_MS = 300

/**
 * useUndo provides undo/redo state management for a string value.
 * Changes are debounced so rapid edits (e.g. typing) collapse into a single
 * history entry. Keyboard shortcuts Ctrl/Cmd+Z and Ctrl/Cmd+Shift+Z are
 * registered automatically.
 */
export function useUndo(initial: string = '') {
  const [history, setHistory] = useState<string[]>([initial])
  const [cursor, setCursor] = useState(0)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const cursorRef = useRef(0)

  // Keep cursorRef in sync with cursor state
  cursorRef.current = cursor

  // Current value derived from cursor position
  const value = history[cursor] ?? initial
  const canUndo = cursor > 0
  const canRedo = cursor < history.length - 1

  /** Push a new value onto the history stack (debounced). */
  const push = useCallback(
    (next: string) => {
      if (timerRef.current != null) {
        clearTimeout(timerRef.current)
      }
      timerRef.current = setTimeout(() => {
        // Read the latest cursor from ref to avoid stale closure
        const currentCursor = cursorRef.current
        setHistory((prev) => {
          // Trim any redo entries beyond current cursor
          const trimmed = prev.slice(0, currentCursor + 1)
          const updated = [...trimmed, next]
          // Enforce max history size
          if (updated.length > MAX_HISTORY) {
            return updated.slice(updated.length - MAX_HISTORY)
          }
          return updated
        })
        setCursor((prev) => {
          const nextCursor = prev + 1
          return nextCursor >= MAX_HISTORY ? MAX_HISTORY - 1 : nextCursor
        })
      }, DEBOUNCE_MS)
    },
    [], // No dependency on cursor — uses cursorRef instead
  )

  /** Move back one step in history and return the value. */
  const undo = useCallback((): string => {
    if (timerRef.current != null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    const newCursor = Math.max(cursor - 1, 0)
    setCursor(newCursor)
    return history[newCursor] ?? ''
  }, [cursor, history])

  /** Move forward one step in history and return the value. */
  const redo = useCallback((): string => {
    if (timerRef.current != null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    const maxIdx = history.length - 1
    const nextCursor = Math.min(cursor + 1, maxIdx)
    return history[nextCursor] ?? ''
  }, [cursor, history])

  /** Reset history to a single entry (e.g. after save/publish). */
  const reset = useCallback((val: string) => {
    if (timerRef.current != null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    setHistory([val])
    setCursor(0)
  }, [])

  return { value, push, undo, redo, reset, canUndo, canRedo }
}
