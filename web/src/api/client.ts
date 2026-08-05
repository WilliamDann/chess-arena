import type { Challenge, Game, Move, Page, Profile } from './types'

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (!res.ok) {
    throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  }
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

const json = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

// Go's json.NewEncoder writes nil slices as `null`, so list endpoints are
// normalized to empty arrays here.
export const api = {
  register: (email: string, password: string, displayName: string) =>
    request<{ id: string; email: string }>(
      '/api/profile',
      json({ email, password, display_name: displayName }),
    ),
  login: (email: string, password: string) => request<unknown>('/api/session', json({ email, password })),
  logout: () => request<void>('/api/session/my', { method: 'DELETE' }),

  myProfile: () => request<Profile>('/api/profile/my'),
  profile: (id: string) => request<Profile>(`/api/profile/${id}`),

  openChallenges: () => request<Challenge[] | null>('/api/challenges/open').then((r) => r ?? []),
  incomingChallenges: () => request<Challenge[] | null>('/api/challenges/in').then((r) => r ?? []),
  outgoingChallenges: () => request<Challenge[] | null>('/api/challenges/out').then((r) => r ?? []),
  createChallenge: (req: { to_player?: string; clock_initial_ms: number; clock_increment_ms: number }) =>
    request<Challenge>('/api/challenge', json(req)),
  deleteChallenge: (id: string) => request<void>(`/api/challenge/${id}`, { method: 'DELETE' }),
  acceptChallenge: (id: string) => request<Game>(`/api/challenge/accept/${id}`, { method: 'POST' }),

  game: (id: string) => request<Game>(`/api/game/${id}`),
  gameMoves: (id: string) => request<Move[] | null>(`/api/game/${id}/moves`).then((r) => r ?? []),
  liveGames: () => request<Game[] | null>('/api/games/live').then((r) => r ?? []),
  userGames: (id: string, limit = 20, offset = 0) =>
    request<Page<Game>>(`/api/games/user/${id}?limit=${limit}&offset=${offset}`),
}
