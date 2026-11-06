import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { LogIn } from 'lucide-react'
import { useAuth } from '../context/AuthContext'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? '/'
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await login(email.trim(), password)
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login gagal')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-md px-4 py-12">
      <div className="card p-6">
        <h1 className="title">Masuk</h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Belum punya akun?{' '}
          <Link to="/register" className="font-bold underline">
            Daftar di sini
          </Link>
        </p>
        <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
          <div>
            <label className="label" htmlFor="email">
              Email
            </label>
            <input
              id="email"
              type="email"
              className="input"
              placeholder="kamu@email.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="label" htmlFor="password">
              Kata Sandi
            </label>
            <input
              id="password"
              type="password"
              className="input"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          {error && (
            <p className="border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
              {error}
            </p>
          )}
          <button type="submit" disabled={busy} className="btn btn-primary w-full">
            <LogIn className="h-4 w-4" /> {busy ? 'Memproses...' : 'Masuk'}
          </button>
        </form>
        <div className="mt-6 border-2 border-dashed border-zinc-400 p-3 text-xs text-zinc-600 dark:text-zinc-400">
          <p className="font-display font-black uppercase">Akun demo</p>
          <p className="mt-1">Customer: customer@ecommerce.local / Customer123!</p>
          <p>Admin: admin@ecommerce.local / Admin123!</p>
        </div>
      </div>
    </div>
  )
}
