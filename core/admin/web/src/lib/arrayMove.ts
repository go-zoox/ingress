/** Swap item at index with its neighbor; returns the same array reference if move is invalid. */
export function moveAdjacent<T>(items: T[], index: number, delta: -1 | 1): T[] {
  const j = index + delta
  if (j < 0 || j >= items.length) return items
  const next = [...items]
  ;[next[index], next[j]] = [next[j], next[index]]
  return next
}

/** Move item from one index to another; returns the same array reference if move is invalid. */
export function moveToIndex<T>(items: T[], from: number, to: number): T[] {
  if (from === to || from < 0 || from >= items.length || to < 0 || to >= items.length) {
    return items
  }
  const next = [...items]
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item)
  return next
}

/** Remap a watched list index after moveToIndex(from, to). */
export function remapIndexAfterMove(
  watchedIndex: number | null,
  from: number,
  to: number,
): number | null {
  if (watchedIndex == null || from === to) return watchedIndex
  if (watchedIndex === from) return to
  if (from < to) {
    if (watchedIndex > from && watchedIndex <= to) return watchedIndex - 1
  } else if (watchedIndex >= to && watchedIndex < from) {
    return watchedIndex + 1
  }
  return watchedIndex
}
