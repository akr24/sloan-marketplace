// Holds the signed-in user and token. The token is kept in localStorage so a
// reload stays signed in; on load we re-check it against /api/auth/me so a
// stale or revoked token is dropped rather than trusted.

import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { auth as authApi, setToken, type Session, type User } from '../api/client'
import { AuthContext, type AuthState } from './context'

const STORAGE_KEY = 'sloanmarket.token'

function storedToken(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  // Only loading if there is a token to verify.
  const [loading, setLoading] = useState(() => storedToken() !== null)

  const accept = useCallback((s: Session) => {
    try {
      localStorage.setItem(STORAGE_KEY, s.token)
    } catch {
      // private mode; stay signed in for this tab only
    }
    setToken(s.token)
    setUser(s.user)
    return s.user
  }, [])

  const logout = useCallback(() => {
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // ignore
    }
    setToken(null)
    setUser(null)
  }, [])

  useEffect(() => {
    const stored = storedToken()
    if (!stored) return
    setToken(stored)
    authApi
      .me()
      .then(setUser)
      .catch(logout)
      .finally(() => setLoading(false))
  }, [logout])

  const value: AuthState = { user, loading, accept, logout }
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
