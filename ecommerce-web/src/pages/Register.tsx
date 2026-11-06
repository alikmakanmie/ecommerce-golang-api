import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { UserPlus } from 'lucide-react'
import { useAuth } from '../context/AuthContext'

export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (password !== confirm) {
      setError('Konfirmasi kata sandi tidak cocok')
      return
    }
    if (password.length < 8) {
      setError('Kata sandi minimal 8 karakter')
      return
    }
    setError('')
    setBusy(true)
    try {
      await register(name.trim(), email.trim(), password, phone.trim() || undefined)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registrasi gagal')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-md px-4 py-12">
      <div className="card p-6">
        <h1 className="title">Daftar</h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Sudah punya akun?{' '}
          <Link to="/login" className="font-bold underline">
            Masuk di sini
          </Link>
        </p>
        <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
          <div>
            <label className="label" htmlFor="name">
              Nama Lengkap
            </label>
            <input
              id="name"
              className="input"
              placeholder="Nama kamu"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              minLength={2}
            />
          </div>
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
            <label className="label" htmlFor="phone">
              No. HP (opsional)
            </label>
            <input
              id="phone"
              type="tel"
              className="input"
              placeholder="0812xxxx"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
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
              placeholder="Minimal 8 karakter"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={8}
            />
          </div>
          <div>
            <label className="label" htmlFor="confirm">
              Konfirmasi Kata Sandi
            </label>
            <input
              id="confirm"
              type="password"
              className="input"
              placeholder="Ulangi kata sandi"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              required
            />
          </div>
          {error && (
            <p className="border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
              {error}
            </p>
          )}
          <button type="submit" disabled={busy} className="btn btn-primary w-full">
            <UserPlus className="h-4 w-4" /> {busy ? 'Memproses...' : 'Buat Akun'}
          </button>
        </form>
      </div>
    </div>
  )
}
