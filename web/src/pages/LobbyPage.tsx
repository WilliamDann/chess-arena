import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useDisplayName } from '../api/profiles'
import { wsUrl } from '../api/ws'
import { useAuth } from '../auth/AuthContext'
import { storeGame } from '../gameCache'
import type { Challenge, Game, LobbyEvent, PresenceEvent, Profile } from '../api/types'

const MINUTES = [1, 2, 3, 5, 10, 15, 30, 60]
const INCREMENTS = [0, 1, 2, 3, 5, 10, 15, 30]

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
  const [outgoing, setOutgoing] = useState<Challenge[]>([])
  const [games, setGames] = useState<Game[]>([])
  const [players, setPlayers] = useState<Profile[]>([])
  const [minutes, setMinutes] = useState(5)
  const [increment, setIncrement] = useState(3)
  const [createError, setCreateError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const loadAll = useCallback(async () => {
    try {
      const [o, out, g, p] = await Promise.all([
        api.openChallenges(),
        api.outgoingChallenges(),
        api.liveGames(),
        api.activePlayers(),
      ])
      setOpen(o)
      setOutgoing(out)
      setGames(g)
      setPlayers(p)
    } catch {
      setNotice('failed to load challenges')
    }
  }, [])

  const handleEvent = useCallback(
    (ev: LobbyEvent | PresenceEvent) => {
      if (ev.type === 'presence.join') {
        setPlayers((l) => (l.some((p) => p.id === ev.profile.id) ? l : [...l, ev.profile]))
      } else if (ev.type === 'presence.leave') {
        setPlayers((l) => l.filter((p) => p.id !== ev.profile.id))
      } else if (ev.type === 'lobby.create') {
        const c = ev.challenge
        if (c.from_player === me) setOutgoing((l) => addUnique(l, c))
        // direct challenges to me surface in the topbar bell, not here
        else if (!c.to_player) setOpen((l) => addUnique(l, c))
      } else if (ev.type === 'lobby.accept') {
        // one of my challenges was accepted: the game exists now, join it
        if (outgoing.some((c) => c.id === ev.id)) {
          storeGame(ev.game)
          navigate(`/game/${ev.game.id}`)
          return
        }
        setOpen((l) => l.filter((c) => c.id !== ev.id))
      } else if (ev.type === 'lobby.delete') {
        setOpen((l) => l.filter((c) => c.id !== ev.id))
        setOutgoing((l) => l.filter((c) => c.id !== ev.id))
      }
    },
    [me, outgoing, navigate],
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
      ws.onmessage = (e) => handleRef.current(JSON.parse(e.data as string) as LobbyEvent | PresenceEvent)
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
    try {
      const c = await api.createChallenge({
        clock_initial_ms: minutes * 60_000,
        clock_increment_ms: increment * 1000,
      })
      setOutgoing((l) => addUnique(l, c))
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : 'failed to create challenge')
    }
  }

  // challenge a specific player, using the time control picked in the form
  const challengePlayer = async (id: string) => {
    try {
      const c = await api.createChallenge({
        to_player: id,
        clock_initial_ms: minutes * 60_000,
        clock_increment_ms: increment * 1000,
      })
      setOutgoing((l) => addUnique(l, c))
      setNotice(null)
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : 'failed to send challenge')
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
    try {
      await api.deleteChallenge(id)
      setOutgoing((l) => l.filter((c) => c.id !== id))
    } catch {
      setNotice('failed to cancel challenge')
    }
  }

  return (
    <div className="lobby">
      {notice && (
        <div className="banner">
          {notice} <button className="ghost" onClick={() => setNotice(null)}>dismiss</button>
        </div>
      )}

      {games.length > 0 && (
        <section className="card">
          <h2>games in play</h2>
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
        </section>
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
            {createError && <p className="error">{createError}</p>}
            <button className="primary">create challenge</button>
          </form>
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
          <h2>players online</h2>
          {players.length === 0 ? (
            <p className="empty">nobody is online</p>
          ) : (
            <ul className="challenge-list">
              {[...players]
                .sort((a, b) => (a.display_name ?? 'anonymous').localeCompare(b.display_name ?? 'anonymous'))
                .map((p) => (
                  <li key={p.id}>
                    <span className="who">
                      <Link to={`/profile/${p.id}`}>{p.display_name ?? 'anonymous'}</Link>
                      {p.id === me && <span className="hint"> (you)</span>}
                    </span>
                    {p.id !== me && (
                      <button
                        className="icon-btn"
                        aria-label={`challenge ${p.display_name ?? 'anonymous'}`}
                        title={`challenge · ${minutes}+${increment}`}
                        onClick={() => void challengePlayer(p.id)}
                      >
                        <SwordIcon />
                      </button>
                    )}
                  </li>
                ))}
            </ul>
          )}
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

// crossed swords, stroke-styled to match the topbar bell icon
function SwordIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <polyline points="14.5 17.5 3 6 3 3 6 3 17.5 14.5" />
      <line x1="13" x2="19" y1="19" y2="13" />
      <line x1="16" x2="20" y1="16" y2="20" />
      <line x1="19" x2="21" y1="21" y2="19" />
      <polyline points="14.5 6.5 18 3 21 3 21 6 17.5 9.5" />
      <line x1="5" x2="9" y1="14" y2="18" />
      <line x1="7" x2="4" y1="17" y2="20" />
      <line x1="3" x2="5" y1="19" y2="21" />
    </svg>
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
      <span className="who">
        <Link to={`/profile/${opponent}`}>{name}</Link>
      </span>
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
      <span className="who">{other ? <Link to={`/profile/${other}`}>{name}</Link> : 'anyone'}</span>
      <button onClick={() => void action.run(challenge.id)}>{action.label}</button>
    </li>
  )
}
