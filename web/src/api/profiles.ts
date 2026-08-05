import { useEffect, useState } from 'react'
import { api } from './client'

// Challenges and games only carry player uuids; display names are resolved
// lazily via GET /api/profile/{id} and cached for the session. Raw ids are
// never surfaced: missing names fall back to "anonymous", '…' while loading.

const cache = new Map<string, string>()
const pending = new Map<string, Promise<string>>()

const FALLBACK = 'anonymous'

function resolve(id: string): Promise<string> {
  let p = pending.get(id)
  if (!p) {
    p = api
      .profile(id)
      .then((profile) => profile.display_name ?? FALLBACK)
      .catch(() => FALLBACK)
      .then((name) => {
        cache.set(id, name)
        pending.delete(id)
        return name
      })
    pending.set(id, p)
  }
  return p
}

export function useDisplayName(id: string | null | undefined): string {
  const [name, setName] = useState(() => (id ? (cache.get(id) ?? '…') : ''))

  useEffect(() => {
    if (!id) return
    if (cache.has(id)) {
      setName(cache.get(id)!)
      return
    }
    setName('…')
    let alive = true
    void resolve(id).then((n) => {
      if (alive) setName(n)
    })
    return () => {
      alive = false
    }
  }, [id])

  return id ? name : ''
}
