import { useEffect, useRef } from 'react'
import { Chessground } from 'chessground'
import type { Api } from 'chessground/api'
import type { Config } from 'chessground/config'
import type { Key } from 'chessground/types'
import 'chessground/assets/chessground.base.css'
import 'chessground/assets/chessground.brown.css'
import 'chessground/assets/chessground.cburnett.css'

type Color = 'white' | 'black'

interface BoardProps {
  fen: string
  orientation: Color
  turnColor: Color
  lastMove?: [Key, Key]
  check: boolean
  viewOnly: boolean
  /** side whose pieces may be picked up; undefined = none */
  movableColor?: Color
  dests: Map<Key, Key[]>
  /** bump to force the board back to `fen` after a rejected drop */
  resetSignal?: number
  onMove: (orig: Key, dest: Key) => void
}

export function Board(props: BoardProps) {
  const { fen, orientation, turnColor, lastMove, check, viewOnly, movableColor, dests, resetSignal, onMove } = props

  const el = useRef<HTMLDivElement>(null)
  const apiRef = useRef<Api | null>(null)
  const onMoveRef = useRef(onMove)
  onMoveRef.current = onMove

  const configRef = useRef<() => Config>(null!)
  configRef.current = () => ({
    fen,
    orientation,
    turnColor,
    lastMove,
    check,
    viewOnly,
    animation: { duration: 150 },
    highlight: { lastMove: true, check: true },
    premovable: { enabled: false },
    draggable: { showGhost: true },
    movable: {
      free: false,
      color: movableColor,
      dests,
      showDests: true,
      events: {
        after: (orig, dest) => onMoveRef.current(orig, dest),
      },
    },
  })

  useEffect(() => {
    apiRef.current = Chessground(el.current!, configRef.current())
    return () => {
      apiRef.current?.destroy()
      apiRef.current = null
    }
  }, [])

  useEffect(() => {
    apiRef.current?.set(configRef.current())
  }, [fen, orientation, turnColor, lastMove, check, viewOnly, movableColor, dests, resetSignal])

  return <div ref={el} className="board" />
}
