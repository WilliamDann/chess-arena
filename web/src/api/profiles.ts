import { useEffect, useState } from 'react'
import { api } from './client'

// Challenges and games only carry player uuids; display names are resolved
// lazily via GET /api/profile/{id} and cached for the session.

const cache = new Map<string, string>()
const pending = new Map<string, Promise<string>>()

export function shortId(id: string): string {
  return id.slice(0, 8)
}

function resolve(id: string): Promise<string> {
  let p = pending.get(id)
  if (!p) {
    p = api
      .profile(id)
      .then((profile) => profile.display_name ?? shortId(id))
      .catch(() => shortId(id))
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
  const [name, setName] = useState(() => (id ? (cache.get(id) ?? shortId(id)) : ''))

  useEffect(() => {
    if (!id) return
    if (cache.has(id)) {
      setName(cache.get(id)!)
      return
    }
    setName(shortId(id))
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
