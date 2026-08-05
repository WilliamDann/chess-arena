import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from '../api/client'
import type { Profile } from '../api/types'

interface Auth {
  profile: Profile | null
  loading: boolean
  login(email: string, password: string): Promise<void>
  register(email: string, password: string, displayName: string): Promise<void>
  logout(): Promise<void>
}

const AuthContext = createContext<Auth | null>(null)

// The session cookie is HttpOnly, so login state can only be observed by
// asking the API who we are.
export function AuthProvider({ children }: { children: ReactNode }) {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      setProfile(await api.myProfile())
    } catch {
      setProfile(null)
    }
  }, [])

  useEffect(() => {
    void refresh().finally(() => setLoading(false))
  }, [refresh])

  const login = useCallback(
    async (email: string, password: string) => {
      await api.login(email, password)
      await refresh()
    },
    [refresh],
  )

  const register = useCallback(
    async (email: string, password: string, displayName: string) => {
      await api.register(email, password, displayName)
      await login(email, password)
    },
    [login],
  )

  const logout = useCallback(async () => {
    try {
      await api.logout()
    } catch {
      // session may already be gone; treat as signed out either way
    }
    setProfile(null)
  }, [])

  return (
    <AuthContext.Provider value={{ profile, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): Auth {
  const auth = useContext(AuthContext)
  if (!auth) throw new Error('useAuth must be used inside AuthProvider')
  return auth
}
