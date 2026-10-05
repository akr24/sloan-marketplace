import { createContext, useContext } from 'react'
import type { Session, User } from '../api/client'

export type AuthState = {
  user: User | null
  loading: boolean
  accept: (s: Session) => User
  logout: () => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
