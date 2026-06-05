import { describe, expect, it } from 'vitest'
import { moveAdjacent, moveToIndex, remapIndexAfterMove } from './arrayMove'

describe('moveAdjacent', () => {
  it('swaps with neighbor', () => {
    expect(moveAdjacent(['a', 'b', 'c'], 1, -1)).toEqual(['b', 'a', 'c'])
    expect(moveAdjacent(['a', 'b', 'c'], 1, 1)).toEqual(['a', 'c', 'b'])
  })

  it('returns same reference when out of bounds', () => {
    const items = ['a']
    expect(moveAdjacent(items, 0, -1)).toBe(items)
    expect(moveAdjacent(items, 0, 1)).toBe(items)
  })
})

describe('moveToIndex', () => {
  it('moves item forward', () => {
    expect(moveToIndex(['a', 'b', 'c'], 0, 2)).toEqual(['b', 'c', 'a'])
  })

  it('moves item backward', () => {
    expect(moveToIndex(['a', 'b', 'c'], 2, 0)).toEqual(['c', 'a', 'b'])
  })

  it('no-ops on invalid indices', () => {
    const items = ['a', 'b']
    expect(moveToIndex(items, 0, 0)).toBe(items)
    expect(moveToIndex(items, -1, 0)).toBe(items)
    expect(moveToIndex(items, 0, 5)).toBe(items)
  })
})

describe('remapIndexAfterMove', () => {
  it('follows the moved row', () => {
    expect(remapIndexAfterMove(1, 1, 3)).toBe(3)
  })

  it('shifts indices when another row moves down', () => {
    expect(remapIndexAfterMove(2, 0, 2)).toBe(1)
  })

  it('shifts indices when another row moves up', () => {
    expect(remapIndexAfterMove(1, 3, 0)).toBe(2)
  })

  it('leaves unrelated indices unchanged', () => {
    expect(remapIndexAfterMove(0, 2, 3)).toBe(0)
    expect(remapIndexAfterMove(null, 1, 3)).toBe(null)
  })
})
