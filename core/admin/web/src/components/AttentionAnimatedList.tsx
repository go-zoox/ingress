import {
  forwardRef,
  Fragment,
  useImperativeHandle,
  type ReactNode,
} from 'react'
import { useAnimatedListRows, type AnimatedListPhase, type AnimatedListRow } from '../hooks/useAnimatedListRows'
import { useListFlip } from '../hooks/useListFlip'

export type AttentionAnimPhase = AnimatedListPhase
export type AttentionAnimRow<T> = AnimatedListRow<T>

export type AttentionAnimatedListHandle = {
  exitThen: (key: string | number, fn: () => void) => void
}

type Props<T> = {
  items: T[]
  itemKey: (item: T) => string | number
  className?: string
  children: (row: AnimatedListRow<T>, layoutRef: (el: HTMLElement | null) => void) => ReactNode
}

export const AttentionAnimatedList = forwardRef(function AttentionAnimatedList<T>(
  { items, itemKey, className, children }: Props<T>,
  ref: React.Ref<AttentionAnimatedListHandle>,
) {
  const { rows, flipKeys, exitThen } = useAnimatedListRows(items, itemKey)
  const registerFlip = useListFlip(flipKeys)

  useImperativeHandle(ref, () => ({ exitThen }), [exitThen])

  return (
    <ul className={className}>
      {rows.map((row) => (
        <Fragment key={row.key}>{children(row, registerFlip(row.key))}</Fragment>
      ))}
    </ul>
  )
}) as <T>(
  props: Props<T> & { ref?: React.Ref<AttentionAnimatedListHandle> },
) => ReactNode
