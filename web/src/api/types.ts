// These shapes mirror the Go structs in src/internal/model — field names must
// match the json tags exactly.

export interface Profile {
  id: string
  display_name: string | null
}

export interface Challenge {
  id: string
  from_player: string
  to_player: string | null
  clock_initial_ms: number
  clock_increment_ms: number
  created_at: string
}

// Game embeds model.Clock, whose json tags differ from Challenge's clock fields
export interface Game {
  id: string
  white_player: string
  black_player: string
  result: string // "*" while undecided, else "1-0" | "0-1" | "1/2-1/2"
  result_at: string | null
  clock_init_ms: number
  clock_inc_ms: number
  created_at: string
}

export interface Move {
  game_id: string
  ply: number // 1-based; odd plies are white's
  uci: string
  created_at: string
  fen: string
  white_ms: number
  black_ms: number
}

export interface Page<T> {
  items: T[]
  limit: number
  offset: number
  total: number
}

export type LobbyEvent =
  | { type: 'lobby.create'; id: string; challenge: Challenge }
  | { type: 'lobby.delete'; id: string }
  // deleted because it was accepted; carries the game it produced
  | { type: 'lobby.accept'; id: string; game: Game }

// delivered on the lobby socket alongside LobbyEvent
export type PresenceEvent =
  | { type: 'presence.join'; profile: Profile }
  | { type: 'presence.leave'; profile: Profile }

export type GameEvent =
  | { type: 'game.move'; move: Move }
  | { type: 'game.end'; game_id: string; result: string; reason: string }
