import { Link, useLocation, useNavigate } from 'react-router-dom'
import { LogOut, Moon, ShoppingCart, Sun, User, UtensilsCrossed } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useCart } from '../context/CartContext'
import { useTheme } from '../context/ThemeContext'
import { useState } from 'react'

export default function Navbar() {
  const { user, isAdmin, logout } = useAuth()
  const { cart } = useCart()
  const { theme, toggle } = useTheme()
  const navigate = useNavigate()
  const location = useLocation()
  const [open, setOpen] = useState(false)

  const totalItems = cart?.total_items ?? 0

  const linkCls = (path: string) =>
    `border-2 px-3 py-1.5 font-display text-xs font-black uppercase tracking-wide transition-all ${
      location.pathname === path
        ? 'border-black bg-black text-white shadow-[3px_3px_0px_0px_#000] dark:border-white dark:bg-white dark:text-black dark:shadow-[3px_3px_0px_0px_#fff]'
        : 'border-transparent hover:border-black dark:hover:border-white'
    }`

  const handleLogout = async () => {
    await logout()
    navigate('/')
  }

  return (
    <header className="sticky top-0 z-40 border-b-4 border-black bg-bone dark:border-white dark:bg-carbon">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-2 px-4 py-3">
        <Link to="/" className="flex items-center gap-2.5">
          <span className="flex h-10 w-10 items-center justify-center border-2 border-black bg-yolk text-black shadow-[3px_3px_0px_0px_#000] dark:border-white dark:shadow-[3px_3px_0px_0px_#fff]">
            <UtensilsCrossed className="h-5 w-5" strokeWidth={2.5} />
          </span>
          <span className="leading-none">
            <span className="block font-display text-lg font-black uppercase tracking-tight sm:text-xl">
              Slaras
            </span>
            <span className="block font-display text-[10px] font-black uppercase tracking-[0.25em] text-zinc-600 dark:text-zinc-400">
              Nusa Foodlab
            </span>
          </span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          <Link to="/" className={linkCls('/')}>
            Beranda
          </Link>
          <Link to="/my-orders" className={linkCls('/my-orders')}>
            Pesanan Saya
          </Link>
          {user && (
            <Link to="/orders" className={linkCls('/orders')}>
              Pesanan
            </Link>
          )}
          {isAdmin && (
            <Link to="/admin" className={linkCls('/admin')}>
              Admin
            </Link>
          )}
        </nav>

        <div className="flex items-center gap-2">
          <button
            onClick={toggle}
            className="btn btn-ghost !px-2.5 !py-2"
            aria-label={theme === 'dark' ? 'Mode terang' : 'Mode gelap'}
            title={theme === 'dark' ? 'Mode terang' : 'Mode gelap'}
          >
            {theme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>

          {user ? (
            <>
              <Link to="/profile" className="btn btn-ghost btn-sm hidden !px-2.5 sm:inline-flex" title={user.name}>
                <User className="h-4 w-4" />
                <span className="max-w-24 truncate">{user.name.split(' ')[0]}</span>
              </Link>
              <button onClick={handleLogout} className="btn btn-ghost btn-sm !px-2.5" title="Keluar">
                <LogOut className="h-4 w-4" />
              </button>
            </>
          ) : (
            <Link to="/login" className="btn btn-primary btn-sm">
              Masuk
            </Link>
          )}

          <Link to="/cart" className="btn btn-accent btn-sm relative !px-3" title="Keranjang">
            <ShoppingCart className="h-4 w-4" />
            <span className="hidden sm:inline">Keranjang</span>
            {totalItems > 0 && (
              <span className="absolute -right-2 -top-2 flex h-5 min-w-5 items-center justify-center border-2 border-black bg-red-500 px-1 font-display text-[10px] font-black text-white dark:border-white">
                {totalItems}
              </span>
            )}
          </Link>

          <button
            className="btn btn-ghost btn-sm !px-2.5 md:hidden"
            onClick={() => setOpen((o) => !o)}
            aria-label="Menu"
          >
            {open ? '✕' : '☰'}
          </button>
        </div>
      </div>

      {open && (
        <nav className="flex flex-col gap-2 border-t-2 border-black px-4 py-3 dark:border-white md:hidden">
          <Link to="/" className={linkCls('/')} onClick={() => setOpen(false)}>
            Beranda
          </Link>
          <Link to="/my-orders" className={linkCls('/my-orders')} onClick={() => setOpen(false)}>
            Pesanan Saya
          </Link>
          {user && (
            <>
              <Link to="/orders" className={linkCls('/orders')} onClick={() => setOpen(false)}>
                Pesanan
              </Link>
              <Link to="/profile" className={linkCls('/profile')} onClick={() => setOpen(false)}>
                Profil & Alamat
              </Link>
            </>
          )}
          {isAdmin && (
            <Link to="/admin" className={linkCls('/admin')} onClick={() => setOpen(false)}>
              Admin
            </Link>
          )}
          {!user && (
            <Link to="/register" className={linkCls('/register')} onClick={() => setOpen(false)}>
              Daftar
            </Link>
          )}
        </nav>
      )}
    </header>
  )
}
