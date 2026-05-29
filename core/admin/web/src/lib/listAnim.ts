import type { AnimatedListPhase } from '../hooks/useAnimatedListRows'

export function listAnimPhaseClass(phase: AnimatedListPhase) {
  if (phase === 'enter') return ' list-anim-enter'
  if (phase === 'exit') return ' list-anim-exit'
  return ''
}
