import { BrowserRouter, Link, Route, Routes } from 'react-router-dom'
import { UtensilsCrossed } from 'lucide-react'
import { AuthProvider } from './context/AuthContext'
import { CartProvider, useCart } from './context/CartContext'
import { ThemeProvider } from './context/ThemeContext'
import Navbar from './components/Navbar'
import FloatingCartBar from './components/FloatingCartBar'
import { ProtectedRoute } from './components/ProtectedRoute'
import Home from './pages/Home'
import ProductDetail from './pages/ProductDetail'
import Login from './pages/Login'
import Register from './pages/Register'
import Cart from './pages/Cart'
import Checkout from './pages/Checkout'
import Orders from './pages/Orders'
import OrderDetail from './pages/OrderDetail'
import TrackOrder from './pages/TrackOrder'
import MyOrders from './pages/MyOrders'
import Profile from './pages/Profile'
import Admin from './pages/Admin'

function Footer() {
  return (
    <footer className="border-t-4 border-black bg-black text-white dark:border-white dark:bg-coal">
      <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-8 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center border-2 border-white bg-yolk text-black">
            <UtensilsCrossed className="h-5 w-5" strokeWidth={2.5} />
          </span>
          <div className="leading-none">
            <p className="font-display text-base font-black uppercase">Slaras Nusa Foodlab</p>
            <p className="text-xs text-zinc-400">Kuliner Nusantara • Dimasak segar setiap hari</p>
          </div>
        </div>
        <nav className="flex flex-wrap gap-4 text-sm font-bold">
          <Link to="/" className="hover:underline">
            Menu
          </Link>
          <Link to="/cart" className="hover:underline">
            Keranjang
          </Link>
          <Link to="/my-orders" className="hover:underline">
            Pesanan Saya
          </Link>
          <Link to="/orders" className="hover:underline">
            Pesanan
          </Link>
          <Link to="/profile" className="hover:underline">
            Profil
          </Link>
        </nav>
        <p className="text-xs text-zinc-400">© 2026 Slaras Nusa Foodlab. Harga dalam Rupiah (IDR).</p>
      </div>
    </footer>
  )
}

function NotFound() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-20 text-center">
      <p className="font-display text-6xl font-black">404</p>
      <p className="mt-2 font-display text-xl font-black uppercase">Halaman tidak ditemukan</p>
      <Link to="/" className="btn btn-primary btn-sm mt-6">
        Kembali ke Beranda
      </Link>
    </div>
  )
}

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <CartProvider>
          <BrowserRouter>
            <AppShell />
          </BrowserRouter>
        </CartProvider>
      </AuthProvider>
    </ThemeProvider>
  )
}

/** Layout dalam: butuh useCart (padding bawah saat floating bar tampil). */
function AppShell() {
  const { cart } = useCart()
  const hasItems = (cart?.total_items ?? 0) > 0
  return (
    <div className="flex min-h-screen flex-col bg-bone text-black dark:bg-carbon dark:text-white">
      <Navbar />
      <main className={`flex-1 ${hasItems ? 'pb-28' : ''}`}>
        <Routes>
                  <Route path="/" element={<Home />} />
                  <Route path="/products/:id" element={<ProductDetail />} />
                  <Route path="/login" element={<Login />} />
                  <Route path="/register" element={<Register />} />
                  {/* Keranjang & checkout publik: tamu bisa pesan tanpa login. */}
                  <Route path="/cart" element={<Cart />} />
                  <Route path="/checkout" element={<Checkout />} />
                  <Route path="/track/:order_number" element={<TrackOrder />} />
                  <Route path="/my-orders" element={<MyOrders />} />
                  <Route
                    path="/orders"
                    element={
                      <ProtectedRoute>
                        <Orders />
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/orders/:id"
                    element={
                      <ProtectedRoute>
                        <OrderDetail />
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/profile"
                    element={
                      <ProtectedRoute>
                        <Profile />
                      </ProtectedRoute>
                    }
                  />
                  <Route
                    path="/admin"
                    element={
                      <ProtectedRoute adminOnly>
                        <Admin />
                      </ProtectedRoute>
                    }
                  />
                  <Route path="*" element={<NotFound />} />
                </Routes>
              </main>
              <Footer />
              <FloatingCartBar />
            </div>
  )
}
