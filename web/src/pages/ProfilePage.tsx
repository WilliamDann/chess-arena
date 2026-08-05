import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { useDisplayName } from '../api/profiles'
import type { Game, Page } from '../api/types'

const PAGE_SIZE = 20

function outcome(g: Game, userId: string): string {
  if (g.result === '*') return 'in play'
  if (g.result === '1/2-1/2') return 'draw'
  const won =
    (g.result === '1-0' && g.white_player === userId) ||
    (g.result === '0-1' && g.black_player === userId)
  return won ? 'win' : 'loss'
}

export function ProfilePage() {
  const { id = '' } = useParams()
  const name = useDisplayName(id)

  const [page, setPage] = useState<Page<Game> | null>(null)
  const [offset, setOffset] = useState(0)
  const [notice, setNotice] = useState<string | null>(null)

  // reset paging when navigating to a different profile
  useEffect(() => setOffset(0), [id])

  useEffect(() => {
    let alive = true
    api
      .userGames(id, PAGE_SIZE, offset)
      .then((p) => {
        if (alive) {
          setPage(p)
          setNotice(null)
        }
      })
      .catch(() => {
        if (alive) setNotice('failed to load games')
      })
    return () => {
      alive = false
    }
  }, [id, offset])

  const games = page?.items ?? []
  const total = page?.total ?? 0

  const share = async () => {
    try {
      await navigator.clipboard.writeText(`${location.origin}/profile/${id}`)
      setNotice('profile link copied — share it to get challenged directly')
    } catch {
      setNotice('could not copy — share this page’s address from the browser bar')
    }
  }

  return (
    <div className="profile">
      <section className="card">
        <h2>{name}</h2>
        <p className="hint">
          {total} game{total === 1 ? '' : 's'}
        </p>
        <button onClick={() => void share()}>copy profile link</button>

        {notice && <p className="notice">{notice}</p>}

        {games.length === 0 ? (
          <p className="empty">no games yet</p>
        ) : (
          <ul className="challenge-list">
            {games.map((g) => (
              <GameHistoryRow key={g.id} game={g} userId={id} />
            ))}
          </ul>
        )}

        {total > PAGE_SIZE && (
          <div className="row">
            <button disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}>
              newer
            </button>
            <span className="hint">
              {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
            </span>
            <button disabled={offset + PAGE_SIZE >= total} onClick={() => setOffset(offset + PAGE_SIZE)}>
              older
            </button>
          </div>
        )}
      </section>
    </div>
  )
}

function GameHistoryRow({ game, userId }: { game: Game; userId: string }) {
  const opponentId = game.white_player === userId ? game.black_player : game.white_player
  const opponent = useDisplayName(opponentId)
  const color = game.white_player === userId ? 'white' : 'black'
  return (
    <li>
      <span className="tc">{`${game.clock_init_ms / 60_000}+${game.clock_inc_ms / 1000}`}</span>
      <span className="who">
        {outcome(game, userId)} as {color} vs{' '}
        <Link to={`/profile/${opponentId}`}>{opponent}</Link>
      </span>
      <Link to={`/game/${game.id}`}>view</Link>
    </li>
  )
}
