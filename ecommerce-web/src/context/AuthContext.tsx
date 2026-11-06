import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { authApi } from '../api/endpoints'
import { apiErrorMessage, tokenStore } from '../api/client'
import type { User } from '../api/types'

interface AuthCtx {
  user: User | null
  loading: boolean
  isAdmin: boolean
  login: (email: string, password: string) => Promise<void>
  register: (name: string, email: string, password: string, phone?: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
}

const Ctx = createContext<AuthCtx>({
  user: null,
  loading: true,
  isAdmin: false,
  login: async () => {},
  register: async () => {},
  logout: async () => {},
  refreshUser: async () => {},
})

function loadStoredUser(): User | null {
  try {
    const raw = localStorage.getItem('snf_user')
    return raw ? (JSON.parse(raw) as User) : null
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => loadStoredUser())
  const [loading, setLoading] = useState(true)

  const refreshUser = useCallback(async () => {
    if (!tokenStore.getAccess()) {
      setUser(null)
      return
    }
    try {
      const me = await authApi.getProfile()
      setUser(me)
      localStorage.setItem('snf_user', JSON.stringify(me))
    } catch {
      tokenStore.clear()
      setUser(null)
    }
  }, [])

  useEffect(() => {
    refreshUser().finally(() => setLoading(false))
  }, [refreshUser])

  const login = useCallback(async (email: string, password: string) => {
    try {
      const payload = await authApi.login(email, password)
      tokenStore.set(payload.access_token, payload.refresh_token)
      setUser(payload.user)
      localStorage.setItem('snf_user', JSON.stringify(payload.user))
    } catch (err) {
      throw new Error(apiErrorMessage(err, 'Login gagal'))
    }
  }, [])

  const register = useCallback(async (name: string, email: string, password: string, phone?: string) => {
    try {
      await authApi.register(name, email, password, phone)
      // Backend register tidak mengembalikan token -> login otomatis.
      await login(email, password)
    } catch (err) {
      throw new Error(apiErrorMessage(err, 'Registrasi gagal'))
    }
  }, [login])

  const logout = useCallback(async () => {
    const refresh = tokenStore.getRefresh()
    if (refresh) {
      try {
        await authApi.logout(refresh)
      } catch {
        // Token sudah tidak valid — tetap bersihkan sesi lokal.
      }
    }
    tokenStore.clear()
    setUser(null)
  }, [])

  return (
    <Ctx.Provider
      value={{
        user,
        loading,
        isAdmin: user?.role === 'admin',
        login,
        register,
        logout,
        refreshUser,
      }}
    >
      {children}
    </Ctx.Provider>
  )
}

export const useAuth = () => useContext(Ctx)
