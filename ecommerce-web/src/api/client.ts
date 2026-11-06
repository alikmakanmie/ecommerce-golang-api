import axios, { type AxiosError, type AxiosRequestConfig } from 'axios'
import type { ApiResponse } from './types'

export const API_URL =
  import.meta.env.VITE_API_URL || 'https://api.alikmakanmie.my.id/api/v1'

const ACCESS_KEY = 'snf_access_token'
const REFRESH_KEY = 'snf_refresh_token'

export const tokenStore = {
  getAccess: () => localStorage.getItem(ACCESS_KEY),
  getRefresh: () => localStorage.getItem(REFRESH_KEY),
  set: (access: string, refresh: string) => {
    localStorage.setItem(ACCESS_KEY, access)
    localStorage.setItem(REFRESH_KEY, refresh)
  },
  clear: () => {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
    localStorage.removeItem('snf_user')
  },
}

export const api = axios.create({
  baseURL: API_URL,
  headers: { 'Content-Type': 'application/json' },
  timeout: 20000,
})

api.interceptors.request.use((config) => {
  const token = tokenStore.getAccess()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

let refreshing: Promise<string> | null = null

async function refreshAccessToken(): Promise<string> {
  if (!refreshing) {
    refreshing = (async () => {
      const refreshToken = tokenStore.getRefresh()
      if (!refreshToken) throw new Error('tidak ada refresh token')
      const res = await axios.post<ApiResponse<{ access_token: string; refresh_token: string }>>(
        `${API_URL}/auth/refresh`,
        { refresh_token: refreshToken },
      )
      const data = res.data.data
      if (!data) throw new Error('refresh gagal')
      tokenStore.set(data.access_token, data.refresh_token)
      return data.access_token
    })()
    try {
      return await refreshing
    } finally {
      refreshing = null
    }
  }
  return refreshing
}

api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as AxiosRequestConfig & { _retry?: boolean }
    const status = error.response?.status
    // 401 -> coba refresh sekali, lalu ulangi request. Refresh endpoint
    // sendiri dan request tanpa token tidak ikut jalur ini.
    if (
      status === 401 &&
      original &&
      !original._retry &&
      !original.url?.includes('/auth/refresh') &&
      !original.url?.includes('/auth/login') &&
      tokenStore.getRefresh()
    ) {
      original._retry = true
      try {
        const newToken = await refreshAccessToken()
        original.headers = { ...original.headers, Authorization: `Bearer ${newToken}` }
        return api(original)
      } catch {
        tokenStore.clear()
        window.location.href = '/login'
        return Promise.reject(error)
      }
    }
    return Promise.reject(error)
  },
)

/** Ambil pesan error backend ({success:false, error:"..."}) secara aman. */
export function apiErrorMessage(err: unknown, fallback = 'Terjadi kesalahan'): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as ApiResponse<unknown> | undefined
    if (data?.error) return data.error
    if (err.response?.status === 0 || !err.response) return 'Tidak dapat terhubung ke server'
    if (err.response?.status === 429) return 'Terlalu banyak permintaan, coba lagi sebentar'
  }
  if (err instanceof Error) return err.message || fallback
  return fallback
}
