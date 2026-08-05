import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Chess, type Square } from 'chess.js'
import type { Key } from 'chessground/types'
import { api } from '../api/client'
import { useDisplayName } from '../api/profiles'
import { wsUrl } from '../api/ws'
import { useAuth } from '../auth/AuthContext'
import { Board } from '../components/Board'
import { loadGame, storeGame } from '../gameCache'
import type { GameEvent, Move } from '../api/types'

type Color = 'white' | 'black'

interface GameOver {
  result: string
  reason: string
}

// The server never reports a rejected move back on the socket (it only logs),
// so an optimistic move that isn't echoed within this window is rolled back.
const REJECT_TIMEOUT_MS = 3000

export function GamePage() {
  const { id = '' } = useParams()
  const { profile } = useAuth()
  const me = profile!.id

  const [game, setGame] = useState(() => loadGame(id))
  const [moves, setMoves] = useState<Move[]>([])
  const [over, setOver] = useState<GameOver | null>(() =>
    game && game.result !== '*' ? { result: game.result, reason: '' } : null,
  )
  const [pending, setPending] = useState<{ ply: number; fen: string; uci: string } | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const [resetSignal, setResetSignal] = useState(0)
  const [confirmResign, setConfirmResign] = useState(false)

  const wsRef = useRef<WebSocket | null>(null)
  const rejectTimer = useRef<number | undefined>(undefined)

  // cache miss (reload, shared link): fetch the game so color, player names,
  // and a finished game's result all survive — for spectators too
  useEffect(() => {
    if (game) return
    let alive = true
    api
      .game(id)
      .then((g) => {
        if (!alive) return
        storeGame(g)
        setGame(g)
        // a game.end event carries a reason, so don't overwrite one
        if (g.result !== '*') setOver((prev) => prev ?? { result: g.result, reason: '' })
      })
      .catch(() => {}) // role stays 'unknown'; the board still works optimistically
    return () => {
      alive = false
    }
  }, [id, game])

  const syncMoves = useCallback(async () => {
    try {
      const fetched = await api.gameMoves(id)
      fetched.sort((a, b) => a.ply - b.ply)
      setMoves((prev) => (fetched.length >= prev.length ? fetched : prev))
    } catch {
      setNotice('failed to load moves for this game')
    }
  }, [id])

  const handleEvent = useCallback(
    (ev: GameEvent) => {
      if (ev.type === 'game.move') {
        window.clearTimeout(rejectTimer.current)
        setPending(null)
        setMoves((prev) => {
          const nextPly = (prev[prev.length - 1]?.ply ?? 0) + 1
          if (ev.move.ply === nextPly) return [...prev, ev.move]
          if (ev.move.ply > nextPly) void syncMoves() // gap: recover from the API
          return prev
        })
      } else if (ev.type === 'game.end') {
        window.clearTimeout(rejectTimer.current)
        setPending(null)
        setOver({ result: ev.result, reason: ev.reason })
      }
    },
    [syncMoves],
  )
  const handleRef = useRef(handleEvent)
  handleRef.current = handleEvent

  useEffect(() => {
    let ws: WebSocket | undefined
    let timer: number | undefined
    let stopped = false
    const connect = () => {
      ws = new WebSocket(wsUrl(`/game/${id}`))
      wsRef.current = ws
      ws.onopen = () => {
        setConnected(true)
        void syncMoves() // recover anything missed while disconnected
      }
      ws.onmessage = (e) => handleRef.current(JSON.parse(e.data as string) as GameEvent)
      ws.onclose = () => {
        setConnected(false)
        if (!stopped) timer = window.setTimeout(connect, 2000)
      }
    }
    connect()
    return () => {
      stopped = true
      window.clearTimeout(timer)
      window.clearTimeout(rejectTimer.current)
      ws?.close()
    }
  }, [id, syncMoves])

  // --- derived position -------------------------------------------------

  const serverLast = moves.length > 0 ? moves[moves.length - 1] : undefined
  const displayFen = pending?.fen ?? serverLast?.fen
  const position = useMemo(() => {
    try {
      return displayFen ? new Chess(displayFen) : new Chess()
    } catch {
      return new Chess()
    }
  }, [displayFen])

  const turnColor: Color = position.turn() === 'w' ? 'white' : 'black'
  const myColor: Color | null = game
    ? game.white_player === me
      ? 'white'
      : game.black_player === me
        ? 'black'
        : null
    : null
  // without the Game object (reload / joined by link) we can't tell whether
  // we're a player, so the board stays optimistic and the server arbitrates
  const role: 'player' | 'spectator' | 'unknown' = game ? (myColor ? 'player' : 'spectator') : 'unknown'

  // a finished game reloaded from the API carries no result, but the final
  // position can still prove checkmate/stalemate
  const inferredOver = useMemo<GameOver | null>(() => {
    if (position.isCheckmate()) return { result: turnColor === 'white' ? '0-1' : '1-0', reason: 'Checkmate' }
    if (position.isStalemate()) return { result: '1/2-1/2', reason: 'Stalemate' }
    return null
  }, [position, turnColor])
  const gameOver = over ?? inferredOver

  const viewOnly = gameOver !== null || role === 'spectator'
  const moveSide: Color | undefined = viewOnly ? undefined : role === 'player' ? myColor! : turnColor
  const active = !viewOnly && pending === null && moveSide === turnColor

  const dests = useMemo(() => {
    const map = new Map<Key, Key[]>()
    if (!active) return map
    for (const m of position.moves({ verbose: true })) {
      const from = m.from as Key
      const arr = map.get(from)
      if (arr) arr.push(m.to as Key)
      else map.set(from, [m.to as Key])
    }
    return map
  }, [position, active])

  const onUserMove = (orig: Key, dest: Key) => {
    const c = new Chess(position.fen())
    let uci = `${orig}${dest}`
    const piece = c.get(orig as Square)
    if (piece?.type === 'p' && (dest.endsWith('8') || dest.endsWith('1'))) uci += 'q' // auto-queen
    try {
      c.move({ from: orig, to: dest, promotion: 'q' })
    } catch {
      setResetSignal((n) => n + 1) // snap the board back
      return
    }
    wsRef.current?.send(JSON.stringify({ type: 'move', uci }))
    setPending({ ply: (serverLast?.ply ?? 0) + 1, fen: c.fen(), uci })
    window.clearTimeout(rejectTimer.current)
    rejectTimer.current = window.setTimeout(() => {
      setPending(null)
      setResetSignal((n) => n + 1)
      setNotice(
        role === 'unknown'
          ? "the server didn't accept that move — you may not be a player in this game"
          : "the server didn't accept that move",
      )
    }, REJECT_TIMEOUT_MS)
  }

  // --- clocks -----------------------------------------------------------

  // clocks only run from ply 3: each side's first move is untimed
  const clocksRunning = gameOver === null && moves.length >= 2
  useTick(clocksRunning)

  const initMs = game?.clock_init_ms ?? serverLast?.white_ms ?? null
  const sideToMove: Color = ((serverLast?.ply ?? 0) + 1) % 2 === 1 ? 'white' : 'black'
  const elapsed = clocksRunning && serverLast ? Date.now() - Date.parse(serverLast.created_at) : 0
  const clockOf = (color: Color): number | null => {
    const base = color === 'white' ? (serverLast?.white_ms ?? initMs) : (serverLast?.black_ms ?? initMs)
    if (base == null) return null
    return Math.max(0, base - (clocksRunning && sideToMove === color ? elapsed : 0))
  }

  const oppColor: Color | null = myColor === 'white' ? 'black' : myColor === 'black' ? 'white' : null
  const canFlag =
    gameOver === null && oppColor !== null && oppColor === sideToMove && moves.length >= 2 && clockOf(oppColor) === 0
  const flag = () => wsRef.current?.send(JSON.stringify({ type: 'flag' }))

  const resign = () => {
    wsRef.current?.send(JSON.stringify({ type: 'resign' }))
    setConfirmResign(false)
  }
  // an unconfirmed resign quietly disarms itself
  useEffect(() => {
    if (!confirmResign) return
    const t = window.setTimeout(() => setConfirmResign(false), 5000)
    return () => window.clearTimeout(t)
  }, [confirmResign])

  // --- presentation -----------------------------------------------------

  const sanMoves = useMemo(() => {
    const c = new Chess()
    return moves.map((m) => {
      try {
        return c.move({ from: m.uci.slice(0, 2), to: m.uci.slice(2, 4), promotion: m.uci[4] }).san
      } catch {
        return m.uci
      }
    })
  }, [moves])
  const moveRows: { n: number; white?: string; black?: string }[] = []
  for (let i = 0; i < sanMoves.length; i += 2) {
    moveRows.push({ n: i / 2 + 1, white: sanMoves[i], black: sanMoves[i + 1] })
  }

  const lastUci = pending?.uci ?? serverLast?.uci
  const lastMove = lastUci ? ([lastUci.slice(0, 2) as Key, lastUci.slice(2, 4) as Key] as [Key, Key]) : undefined

  const whiteName = useDisplayName(game?.white_player)
  const blackName = useDisplayName(game?.black_player)
  const nameOf = (color: Color) => {
    const name = color === 'white' ? whiteName : blackName
    return name || color
  }

  const orientation: Color = myColor ?? 'white'
  const topColor: Color = orientation === 'white' ? 'black' : 'white'
  const bottomColor: Color = orientation

  const share = async () => {
    const link = `${location.origin}/game/${id}`
    try {
      await navigator.clipboard.writeText(link)
      setNotice('game link copied — send it to your opponent')
    } catch {
      setNotice(`game link: ${link}`)
    }
  }

  return (
    <div className="game">
      <div className="board-col">
        <PlayerBar name={nameOf(topColor)} ms={clockOf(topColor)} active={gameOver === null && sideToMove === topColor} />
        <Board
          fen={position.fen()}
          orientation={orientation}
          turnColor={turnColor}
          lastMove={lastMove}
          check={position.inCheck()}
          viewOnly={viewOnly}
          movableColor={moveSide}
          dests={dests}
          resetSignal={resetSignal}
          onMove={onUserMove}
        />
        <PlayerBar
          name={nameOf(bottomColor)}
          ms={clockOf(bottomColor)}
          active={gameOver === null && sideToMove === bottomColor}
        />
      </div>

      <aside className="card game-side">
        <div className="status-line">
          <span className={connected ? 'dot on' : 'dot'} />
          {connected ? 'live' : 'reconnecting…'}
          {role === 'spectator' && ' · spectating'}
          {role === 'player' && ` · you play ${myColor}`}
        </div>

        {gameOver && (
          <div className="result">
            {gameOver.result}
            {gameOver.reason && ` · ${gameOver.reason}`}
          </div>
        )}
        {canFlag && (
          <button className="primary" onClick={flag}>
            claim win on time
          </button>
        )}
        {role === 'player' &&
          gameOver === null &&
          (confirmResign ? (
            <div className="row">
              <button className="primary" onClick={resign}>
                confirm resign
              </button>
              <button onClick={() => setConfirmResign(false)}>cancel</button>
            </div>
          ) : (
            <button disabled={!connected} onClick={() => setConfirmResign(true)}>
              resign
            </button>
          ))}

        {moveRows.length > 0 ? (
          <table className="move-table">
            <tbody>
              {moveRows.map((r) => (
                <tr key={r.n}>
                  <td className="num">{r.n}.</td>
                  <td>{r.white}</td>
                  <td>{r.black}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="empty">no moves yet</p>
        )}

        <button onClick={() => void share()}>copy game link</button>
        {role === 'unknown' && (
          <p className="hint">
            joined by link, so your side isn't known — if you're a player, your moves will count when
            it's your turn.
          </p>
        )}
        {notice && (
          <p className="notice">
            {notice}{' '}
            <button className="ghost" onClick={() => setNotice(null)}>
              dismiss
            </button>
          </p>
        )}
      </aside>
    </div>
  )
}

// re-render on an interval while a clock is counting down
function useTick(active: boolean) {
  const [, setTick] = useState(0)
  useEffect(() => {
    if (!active) return
    const t = window.setInterval(() => setTick((n) => n + 1), 200)
    return () => window.clearInterval(t)
  }, [active])
}

function PlayerBar({ name, ms, active }: { name: string; ms: number | null; active: boolean }) {
  return (
    <div className={active ? 'player-bar active' : 'player-bar'}>
      <span className="player-name">{name}</span>
      <span className="clock">{ms == null ? '--:--' : formatClock(ms)}</span>
    </div>
  )
}

function formatClock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}:${String(s).padStart(2, '0')}`
}
