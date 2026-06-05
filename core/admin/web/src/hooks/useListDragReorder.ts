import { useCallback, useState, type DragEvent } from 'react'
import { moveToIndex } from '../lib/arrayMove'

type UseListDragReorderOptions<T> = {
  items: T[]
  onMove: (from: number, to: number) => void
}

export function useListDragReorder<T>({ items, onMove }: UseListDragReorderOptions<T>) {
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [overIndex, setOverIndex] = useState<number | null>(null)

  const clear = useCallback(() => {
    setDragIndex(null)
    setOverIndex(null)
  }, [])

  const rowClassName = useCallback(
    (index: number, extra?: string) => {
      const parts: string[] = []
      if (extra) parts.push(extra)
      if (dragIndex === index) parts.push('row-dragging')
      else if (overIndex === index && dragIndex != null && dragIndex !== index) {
        parts.push('row-drag-over')
      }
      return parts.length > 0 ? parts.join(' ') : undefined
    },
    [dragIndex, overIndex],
  )

  const handleProps = useCallback(
    (index: number) => ({
      draggable: true,
      'aria-grabbed': dragIndex === index,
      onDragStart: (e: DragEvent) => {
        setDragIndex(index)
        setOverIndex(index)
        e.dataTransfer.effectAllowed = 'move'
        e.dataTransfer.setData('text/plain', String(index))
      },
      onDragEnd: () => clear(),
    }),
    [clear, dragIndex],
  )

  const rowProps = useCallback(
    (index: number) => ({
      onDragOver: (e: DragEvent) => {
        e.preventDefault()
        e.dataTransfer.dropEffect = 'move'
        if (dragIndex != null && index !== overIndex) setOverIndex(index)
      },
      onDragLeave: (e: DragEvent) => {
        if (e.currentTarget.contains(e.relatedTarget as Node)) return
        if (overIndex === index) setOverIndex(null)
      },
      onDrop: (e: DragEvent) => {
        e.preventDefault()
        const raw = e.dataTransfer.getData('text/plain')
        const from = dragIndex ?? (raw === '' ? NaN : Number(raw))
        if (!Number.isFinite(from) || from === index) {
          clear()
          return
        }
        if (moveToIndex(items, from, index) !== items) {
          onMove(from, index)
        }
        clear()
      },
    }),
    [clear, dragIndex, items, onMove, overIndex],
  )

  return { dragIndex, rowClassName, handleProps, rowProps }
}
