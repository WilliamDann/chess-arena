import type { Game } from './api/types'

// The Game object is stashed here when navigating from the lobby so the game
// page can render player info without waiting a round trip; on a cache miss
// (reload, shared link) the page fetches GET /api/game/{id} instead.

const key = (id: string) => `chess-arena-game-${id}`

export function storeGame(game: Game): void {
  sessionStorage.setItem(key(game.id), JSON.stringify(game))
}

export function loadGame(id: string): Game | null {
  const raw = sessionStorage.getItem(key(id))
  if (!raw) return null
  try {
    return JSON.parse(raw) as Game
  } catch {
    return null
  }
}
