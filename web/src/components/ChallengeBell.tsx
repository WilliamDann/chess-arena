import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useDisplayName } from '../api/profiles'
import { wsUrl } from '../api/ws'
import { useAuth } from '../auth/AuthContext'
import { storeGame } from '../gameCache'
import type { Challenge, LobbyEvent, PresenceEvent } from '../api/types'

// Topbar bell tracking challenges sent directly to me. It holds its own
// lobby socket so notifications arrive on every page, not just the lobby.
export function ChallengeBell() {
  const { profile } = useAuth()
  const me = profile!.id
  const navigate = useNavigate()

  const [incoming, setIncoming] = useState<Challenge[]>([])
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement | null>(null)

  const load = useCallback(async () => {
    try {
      setIncoming(await api.incomingChallenges())
    } catch {
      // keep the last known list; the next reconnect retries
    }
  }, [])

  const handleEvent = useCallback(
    (ev: LobbyEvent | PresenceEvent) => {
      if (ev.type === 'lobby.create') {
        const c = ev.challenge
        if (c.to_player === me) {
          setIncoming((l) => (l.some((x) => x.id === c.id) ? l : [...l, c]))
          setOpen(true) // surface fresh challenges without waiting for a click
        }
      } else if (ev.type === 'lobby.delete' || ev.type === 'lobby.accept') {
        setIncoming((l) => l.filter((c) => c.id !== ev.id))
      }
    },
    [me],
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
      ws.onopen = () => void load()
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
  }, [load])

  // close the menu on any click outside it
  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open])

  const dismiss = async (id: string) => {
    try {
      await api.deleteChallenge(id)
      setIncoming((l) => l.filter((c) => c.id !== id))
    } catch {
      void load() // already gone or not ours; resync
    }
  }

  const clearAll = async () => {
    try {
      await api.clearIncomingChallenges()
      setIncoming([])
      setOpen(false)
    } catch {
      void load() // partial or failed clear; resync with the server
    }
  }

  const accept = async (id: string) => {
    try {
      const game = await api.acceptChallenge(id)
      setOpen(false)
      storeGame(game)
      navigate(`/game/${game.id}`)
    } catch {
      void load() // withdrawn or already accepted; resync
    }
  }

  return (
    <div className="bell" ref={rootRef}>
      <button
        className="ghost bell-button"
        aria-label={`incoming challenges (${incoming.length})`}
        onClick={() => setOpen((o) => !o)}
      >
        <BellIcon />
        {incoming.length > 0 && <span className="bell-badge">{incoming.length}</span>}
      </button>
      {open && (
        <div className="bell-menu card">
          {incoming.length === 0 ? (
            <p className="empty"></p>
          ) : (
            <>
              <ul className="challenge-list">
                {incoming.map((c) => (
                  <BellRow
                    key={c.id}
                    challenge={c}
                    accept={() => void accept(c.id)}
                    dismiss={() => void dismiss(c.id)}
                    onNav={() => setOpen(false)}
                  />
                ))}
              </ul>
              <button className="ghost bell-clear" onClick={() => void clearAll()}>
                clear all
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}

function BellRow({
  challenge,
  accept,
  dismiss,
  onNav,
}: {
  challenge: Challenge
  accept: () => void
  dismiss: () => void
  onNav: () => void
}) {
  const name = useDisplayName(challenge.from_player)
  return (
    <li>
      <span className="tc">{`${challenge.clock_initial_ms / 60_000}+${challenge.clock_increment_ms / 1000}`}</span>
      <span className="who">
        <Link to={`/profile/${challenge.from_player}`} onClick={onNav}>
          {name}
        </Link>
      </span>
      <button onClick={accept}>accept</button>
      <button className="ghost" onClick={dismiss}>
        dismiss
      </button>
    </li>
  )
}

function BellIcon() {
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
      <path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
      <path d="M13.73 21a2 2 0 0 1-3.46 0" />
    </svg>
  )
}
