import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { MapPin, Plus, Star, Trash2 } from 'lucide-react'
import { addressApi, authApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Address, AddressInput } from '../api/types'
import { useAuth } from '../context/AuthContext'
import { Spinner } from '../components/ProtectedRoute'

const EMPTY_FORM: AddressInput = {
  label: '',
  recipient_name: '',
  phone: '',
  street: '',
  city: '',
  province: '',
  postal_code: '',
  is_default: false,
}

export default function Profile() {
  const { user, refreshUser } = useAuth()
  const [name, setName] = useState(user?.name ?? '')
  const [phone, setPhone] = useState(user?.phone ?? '')
  const [profileMsg, setProfileMsg] = useState('')
  const [profileErr, setProfileErr] = useState('')
  const [savingProfile, setSavingProfile] = useState(false)

  const [addresses, setAddresses] = useState<Address[]>([])
  const [loadingAddr, setLoadingAddr] = useState(true)
  const [form, setForm] = useState<AddressInput>(EMPTY_FORM)
  const [addrErr, setAddrErr] = useState('')
  const [savingAddr, setSavingAddr] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  const fetchAddresses = useCallback(async () => {
    setLoadingAddr(true)
    try {
      setAddresses(await addressApi.list())
    } catch {
      setAddresses([])
    } finally {
      setLoadingAddr(false)
    }
  }, [])

  useEffect(() => {
    fetchAddresses()
  }, [fetchAddresses])

  useEffect(() => {
    if (user) {
      setName(user.name)
      setPhone(user.phone)
    }
  }, [user])

  const saveProfile = async (e: FormEvent) => {
    e.preventDefault()
    setProfileErr('')
    setProfileMsg('')
    setSavingProfile(true)
    try {
      await authApi.updateProfile(name.trim(), phone.trim())
      await refreshUser()
      setProfileMsg('Profil berhasil diperbarui')
    } catch (err) {
      setProfileErr(apiErrorMessage(err, 'Gagal memperbarui profil'))
    } finally {
      setSavingProfile(false)
    }
  }

  const saveAddress = async (e: FormEvent) => {
    e.preventDefault()
    setAddrErr('')
    setSavingAddr(true)
    try {
      await addressApi.create(form)
      setForm(EMPTY_FORM)
      await fetchAddresses()
    } catch (err) {
      setAddrErr(apiErrorMessage(err, 'Gagal menyimpan alamat'))
    } finally {
      setSavingAddr(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm('Hapus alamat ini?')) return
    setBusyId(id)
    try {
      await addressApi.remove(id)
      await fetchAddresses()
    } catch (err) {
      setAddrErr(apiErrorMessage(err, 'Gagal menghapus alamat'))
    } finally {
      setBusyId(null)
    }
  }

  const handleSetDefault = async (id: string) => {
    setBusyId(id)
    try {
      await addressApi.setDefault(id)
      await fetchAddresses()
    } catch (err) {
      setAddrErr(apiErrorMessage(err, 'Gagal mengatur alamat default'))
    } finally {
      setBusyId(null)
    }
  }

  if (!user) return <Spinner label="Memuat profil..." />

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="title">Profil Saya</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {user.email} • Peran: <strong className="uppercase">{user.role}</strong>
      </p>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <section className="card h-fit p-5">
          <h2 className="font-display text-lg font-black uppercase">Edit Profil</h2>
          <form onSubmit={saveProfile} className="mt-4 flex flex-col gap-4">
            <div>
              <label className="label" htmlFor="pname">
                Nama
              </label>
              <input
                id="pname"
                className="input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                maxLength={100}
              />
            </div>
            <div>
              <label className="label" htmlFor="pphone">
                No. HP
              </label>
              <input
                id="pphone"
                className="input"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                maxLength={20}
              />
            </div>
            <div>
              <label className="label">Email</label>
              <input className="input opacity-60" value={user.email} disabled />
            </div>
            {profileMsg && <p className="text-sm font-semibold text-green-700">{profileMsg}</p>}
            {profileErr && <p className="text-sm font-semibold text-red-600">{profileErr}</p>}
            <button type="submit" disabled={savingProfile} className="btn btn-primary w-full">
              {savingProfile ? 'Menyimpan...' : 'Simpan Profil'}
            </button>
          </form>
        </section>

        <section className="card h-fit p-5">
          <h2 className="flex items-center gap-2 font-display text-lg font-black uppercase">
            <MapPin className="h-5 w-5" /> Alamat Pengiriman
          </h2>
          {loadingAddr ? (
            <Spinner label="Memuat alamat..." />
          ) : addresses.length === 0 ? (
            <p className="mt-3 text-sm text-zinc-500">Belum ada alamat. Tambah di bawah.</p>
          ) : (
            <ul className="mt-4 flex flex-col gap-3">
              {addresses.map((a) => (
                <li key={a.id} className="border-2 border-black p-3 dark:border-white">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <p className="font-display text-sm font-black uppercase">
                        {a.label}{' '}
                        {a.is_default && (
                          <span className="chip bg-yolk !text-black">
                            <Star className="h-3 w-3" /> Default
                          </span>
                        )}
                      </p>
                      <p className="text-sm font-bold">
                        {a.recipient_name} • {a.phone}
                      </p>
                      <p className="text-sm text-zinc-600 dark:text-zinc-400">
                        {a.street}, {a.city}, {a.province} {a.postal_code}
                      </p>
                    </div>
                    <div className="flex shrink-0 gap-1">
                      {!a.is_default && (
                        <button
                          onClick={() => handleSetDefault(a.id)}
                          disabled={busyId === a.id}
                          className="btn btn-ghost btn-sm !px-2"
                          title="Jadikan default"
                        >
                          <Star className="h-4 w-4" />
                        </button>
                      )}
                      <button
                        onClick={() => handleDelete(a.id)}
                        disabled={busyId === a.id}
                        className="btn btn-ghost btn-sm !px-2 !text-red-600"
                        title="Hapus alamat"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}

          <form onSubmit={saveAddress} className="mt-5 flex flex-col gap-3 border-t-2 border-black pt-4 dark:border-white">
            <p className="font-display text-sm font-black uppercase">+ Tambah Alamat Baru</p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="label" htmlFor="label">
                  Label
                </label>
                <input
                  id="label"
                  className="input"
                  placeholder="Rumah / Kantor"
                  value={form.label}
                  onChange={(e) => setForm({ ...form, label: e.target.value })}
                  required
                  maxLength={50}
                />
              </div>
              <div>
                <label className="label" htmlFor="recipient">
                  Nama Penerima
                </label>
                <input
                  id="recipient"
                  className="input"
                  value={form.recipient_name}
                  onChange={(e) => setForm({ ...form, recipient_name: e.target.value })}
                  required
                  maxLength={100}
                />
              </div>
            </div>
            <div>
              <label className="label" htmlFor="aphone">
                No. HP
              </label>
              <input
                id="aphone"
                className="input"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
                required
                maxLength={20}
              />
            </div>
            <div>
              <label className="label" htmlFor="street">
                Jalan
              </label>
              <input
                id="street"
                className="input"
                placeholder="Jl. Contoh No. 12"
                value={form.street}
                onChange={(e) => setForm({ ...form, street: e.target.value })}
                required
                maxLength={255}
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="label" htmlFor="city">
                  Kota
                </label>
                <input
                  id="city"
                  className="input"
                  value={form.city}
                  onChange={(e) => setForm({ ...form, city: e.target.value })}
                  required
                  maxLength={100}
                />
              </div>
              <div>
                <label className="label" htmlFor="province">
                  Provinsi
                </label>
                <input
                  id="province"
                  className="input"
                  value={form.province}
                  onChange={(e) => setForm({ ...form, province: e.target.value })}
                  required
                  maxLength={100}
                />
              </div>
              <div>
                <label className="label" htmlFor="postal">
                  Kode Pos
                </label>
                <input
                  id="postal"
                  className="input"
                  value={form.postal_code}
                  onChange={(e) => setForm({ ...form, postal_code: e.target.value })}
                  required
                  maxLength={10}
                />
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm font-semibold">
              <input
                type="checkbox"
                className="h-4 w-4 accent-black"
                checked={form.is_default}
                onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
              />
              Jadikan alamat default
            </label>
            {addrErr && <p className="text-sm font-semibold text-red-600">{addrErr}</p>}
            <button type="submit" disabled={savingAddr} className="btn btn-accent w-full">
              <Plus className="h-4 w-4" /> {savingAddr ? 'Menyimpan...' : 'Simpan Alamat'}
            </button>
          </form>
        </section>
      </div>
    </div>
  )
}
