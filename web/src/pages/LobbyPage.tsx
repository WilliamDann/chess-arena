import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useDisplayName } from '../api/profiles'
import { wsUrl } from '../api/ws'
import { useAuth } from '../auth/AuthContext'
import { storeGame } from '../gameCache'
import type { Challenge, Game, LobbyEvent } from '../api/types'

const MINUTES = [1, 2, 3, 5, 10, 15, 30, 60]
const INCREMENTS = [0, 1, 2, 3, 5, 10, 15, 30]
const UUID_RE = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i

function timeControl(c: Challenge): string {
  return `${c.clock_initial_ms / 60_000}+${c.clock_increment_ms / 1000}`
}

function addUnique(list: Challenge[], c: Challenge): Challenge[] {
  return list.some((x) => x.id === c.id) ? list : [...list, c]
}

export function LobbyPage() {
  const { profile } = useAuth()
  const me = profile!.id
  const navigate = useNavigate()

  const [open, setOpen] = useState<Challenge[]>([])
  const [incoming, setIncoming] = useState<Challenge[]>([])
  const [outgoing, setOutgoing] = useState<Challenge[]>([])
  const [games, setGames] = useState<Game[]>([])
  // game ids already known, so an accepted challenge's *new* game can be told apart
  const knownGames = useRef(new Set<string>())
  const [minutes, setMinutes] = useState(5)
  const [increment, setIncrement] = useState(3)
  const [opponent, setOpponent] = useState('')
  const [joinId, setJoinId] = useState('')
  const [createError, setCreateError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [accepted, setAccepted] = useState(false)
  // ids we deleted ourselves, so their lobby.delete events don't read as "accepted"
  const cancelled = useRef(new Set<string>())

  const loadAll = useCallback(async () => {
    try {
      const [o, i, out, g] = await Promise.all([
        api.openChallenges(),
        api.incomingChallenges(),
        api.outgoingChallenges(),
        api.liveGames(),
      ])
      setOpen(o)
      setIncoming(i)
      setOutgoing(out)
      setGames(g)
      for (const game of g) knownGames.current.add(game.id)
    } catch {
      setNotice('failed to load challenges')
    }
  }, [])

  // The accept flow publishes the lobby.delete event before the game row is
  // committed, so the new game is polled for briefly before giving up.
  const goToNewGame = useCallback(async () => {
    for (let attempt = 0; attempt < 5; attempt++) {
      try {
        const live = await api.liveGames()
        const fresh = live.find((g) => !knownGames.current.has(g.id))
        if (fresh) {
          storeGame(fresh)
          navigate(`/game/${fresh.id}`)
          return
        }
      } catch {
        // transient — keep polling
      }
      await new Promise((r) => setTimeout(r, 400))
    }
    setAccepted(true) // fall back to telling the user where to look
    void loadAll()
  }, [navigate, loadAll])

  const handleEvent = useCallback(
    (ev: LobbyEvent) => {
      if (ev.type === 'lobby.create') {
        const c = ev.challenge
        if (c.from_player === me) setOutgoing((l) => addUnique(l, c))
        else if (c.to_player === me) setIncoming((l) => addUnique(l, c))
        else if (!c.to_player) setOpen((l) => addUnique(l, c))
      } else if (ev.type === 'lobby.delete') {
        setOpen((l) => l.filter((c) => c.id !== ev.id))
        setIncoming((l) => l.filter((c) => c.id !== ev.id))
        setOutgoing((l) => {
          if (l.some((c) => c.id === ev.id) && !cancelled.current.has(ev.id)) void goToNewGame()
          return l.filter((c) => c.id !== ev.id)
        })
      }
    },
    [me, goToNewGame],
  )
  const handleRef = useRef(handleEvent)
  handleRef.current = handleEvent

  useEffect(() => {
    let ws: WebSocket | undefined
    let timer: number | undefined
    let stopped = false
    const connect = () => {
      ws = new WebSocket(wsUrl('/lobby'))
      // reload on (re)connect so events missed while disconnected aren't lost
      ws.onopen = () => void loadAll()
      ws.onmessage = (e) => handleRef.current(JSON.parse(e.data as string) as LobbyEvent)
      ws.onclose = () => {
        if (!stopped) timer = window.setTimeout(connect, 2000)
      }
    }
    connect()
    return () => {
      stopped = true
      window.clearTimeout(timer)
      ws?.close()
    }
  }, [loadAll])

  const create = async (e: FormEvent) => {
    e.preventDefault()
    setCreateError(null)
    const to = opponent.trim()
    if (to && !UUID_RE.test(to)) {
      setCreateError('opponent must be a player id (uuid)')
      return
    }
    try {
      const c = await api.createChallenge({
        to_player: to || undefined,
        clock_initial_ms: minutes * 60_000,
        clock_increment_ms: increment * 1000,
      })
      setOutgoing((l) => addUnique(l, c))
      setOpponent('')
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : 'failed to create challenge')
    }
  }

  const accept = async (id: string) => {
    try {
      const game = await api.acceptChallenge(id)
      storeGame(game)
      navigate(`/game/${game.id}`)
    } catch {
      setNotice('could not accept — the challenge may have been withdrawn')
      void loadAll()
    }
  }

  const cancel = async (id: string) => {
    cancelled.current.add(id)
    try {
      await api.deleteChallenge(id)
      setOutgoing((l) => l.filter((c) => c.id !== id))
    } catch {
      cancelled.current.delete(id)
      setNotice('failed to cancel challenge')
    }
  }

  const join = (e: FormEvent) => {
    e.preventDefault()
    const m = joinId.match(UUID_RE)
    if (m) navigate(`/game/${m[0].toLowerCase()}`)
  }

  return (
    <div className="lobby">
      {accepted && (
        <div className="banner">
          <strong>one of your challenges was just accepted</strong> but the game hasn't appeared
          yet — it should show up under “games in play” in a moment.{' '}
          <button className="ghost" onClick={() => setAccepted(false)}>dismiss</button>
        </div>
      )}
      {notice && (
        <div className="banner">
          {notice} <button className="ghost" onClick={() => setNotice(null)}>dismiss</button>
        </div>
      )}

      <div className="lobby-grid">
        <section className="card">
          <h2>new challenge</h2>
          <form onSubmit={create} className="stack">
            <div className="row">
              <label>
                minutes
                <select value={minutes} onChange={(e) => setMinutes(Number(e.target.value))}>
                  {MINUTES.map((m) => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                </select>
              </label>
              <label>
                increment (s)
                <select value={increment} onChange={(e) => setIncrement(Number(e.target.value))}>
                  {INCREMENTS.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </label>
            </div>
            <label>
              opponent id <span className="hint">(blank = open to anyone)</span>
              <input value={opponent} onChange={(e) => setOpponent(e.target.value)} placeholder="optional uuid" />
            </label>
            {createError && <p className="error">{createError}</p>}
            <button className="primary">create challenge</button>
          </form>

          <hr />

          <form onSubmit={join} className="stack">
            <label>
              join a game <span className="hint">(paste a game link or id)</span>
              <input value={joinId} onChange={(e) => setJoinId(e.target.value)} placeholder="game link or uuid" />
            </label>
            <button disabled={!UUID_RE.test(joinId)}>join</button>
          </form>

          <p className="hint">
            your player id: <code className="selectable">{me}</code>
          </p>
        </section>

        <section className="card">
          <h2>games in play</h2>
          {games.length === 0 ? (
            <p className="empty">no games in play</p>
          ) : (
            <ul className="challenge-list">
              {games.map((g) => (
                <GameRow
                  key={g.id}
                  game={g}
                  me={me}
                  resume={() => {
                    storeGame(g)
                    navigate(`/game/${g.id}`)
                  }}
                />
              ))}
            </ul>
          )}
        </section>

        <section className="card">
          <h2>open challenges</h2>
          <ChallengeList
            items={open.filter((c) => c.from_player !== me)}
            empty="no open challenges — create one"
            action={{ label: 'accept', run: accept }}
            who="from"
          />
        </section>

        <section className="card">
          <h2>incoming</h2>
          <ChallengeList
            items={incoming}
            empty="nobody has challenged you"
            action={{ label: 'accept', run: accept }}
            who="from"
          />
        </section>

        <section className="card">
          <h2>your challenges</h2>
          <ChallengeList
            items={outgoing}
            empty="none pending"
            action={{ label: 'cancel', run: cancel }}
            who="to"
          />
        </section>
      </div>
    </div>
  )
}

interface RowAction {
  label: string
  run: (id: string) => void | Promise<void>
}

function GameRow({ game, me, resume }: { game: Game; me: string; resume: () => void }) {
  const opponent = game.white_player === me ? game.black_player : game.white_player
  const name = useDisplayName(opponent)
  return (
    <li>
      <span className="tc">{`${game.clock_init_ms / 60_000}+${game.clock_inc_ms / 1000}`}</span>
      <span className="who">{name}</span>
      <button onClick={resume}>resume</button>
    </li>
  )
}

function ChallengeList({
  items,
  empty,
  action,
  who,
}: {
  items: Challenge[]
  empty: string
  action: RowAction
  who: 'from' | 'to'
}) {
  if (items.length === 0) return <p className="empty">{empty}</p>
  return (
    <ul className="challenge-list">
      {items.map((c) => (
        <ChallengeRow key={c.id} challenge={c} action={action} who={who} />
      ))}
    </ul>
  )
}

function ChallengeRow({
  challenge,
  action,
  who,
}: {
  challenge: Challenge
  action: RowAction
  who: 'from' | 'to'
}) {
  const other = who === 'from' ? challenge.from_player : challenge.to_player
  const name = useDisplayName(other)
  return (
    <li>
      <span className="tc">{timeControl(challenge)}</span>
      <span className="who">{other ? name : 'anyone'}</span>
      <button onClick={() => void action.run(challenge.id)}>{action.label}</button>
    </li>
  )
}
