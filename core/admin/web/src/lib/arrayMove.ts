/** Swap item at index with its neighbor; returns the same array reference if move is invalid. */
export function moveAdjacent<T>(items: T[], index: number, delta: -1 | 1): T[] {
  const j = index + delta
  if (j < 0 || j >= items.length) return items
  const next = [...items]
  ;[next[index], next[j]] = [next[j], next[index]]
  return next
}
