import { BrowserRouter, Navigate, NavLink, Route, Routes, useLocation } from 'react-router-dom'
import './App.css'
import { AuthProvider } from './auth/AuthContext'
import { useAuth } from './auth/context'
import AccountPage from './pages/AccountPage'
import BrowsePage from './pages/BrowsePage'
import EditListingPage from './pages/EditListingPage'
import ListingPage from './pages/ListingPage'
import MyListingsPage from './pages/MyListingsPage'
import SellPage from './pages/SellPage'
import SignInPage from './pages/SignInPage'

// Phone-first: a bottom tab bar within thumb reach, like a native app.
function TabBar() {
  return (
    <nav className="tabbar" aria-label="Main">
      <NavLink to="/" end>
        <span aria-hidden>🎟️</span>Browse
      </NavLink>
      <NavLink to="/sell">
        <span aria-hidden>➕</span>Sell
      </NavLink>
      <NavLink to="/mine">
        <span aria-hidden>📋</span>My listings
      </NavLink>
      <NavLink to="/account">
        <span aria-hidden>👤</span>Account
      </NavLink>
    </nav>
  )
}

// Listings carry sellers' phone numbers, so everything but sign-in is
// members-only. After signing in the user lands back where they were headed.
function RequireAuth({ children }: { children: React.ReactElement }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <p className="center muted">Loading…</p>
  if (!user) return <Navigate to="/signin" replace state={{ from: location.pathname }} />
  return (
    <>
      <main>{children}</main>
      <TabBar />
    </>
  )
}

function GuestOnly({ children }: { children: React.ReactElement }) {
  const { user, loading } = useAuth()
  if (loading) return null
  return user ? <Navigate to="/" replace /> : <main>{children}</main>
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route
            path="/signin"
            element={
              <GuestOnly>
                <SignInPage />
              </GuestOnly>
            }
          />
          <Route path="/" element={<RequireAuth><BrowsePage /></RequireAuth>} />
          <Route path="/listings/:id" element={<RequireAuth><ListingPage /></RequireAuth>} />
          <Route path="/listings/:id/edit" element={<RequireAuth><EditListingPage /></RequireAuth>} />
          <Route path="/sell" element={<RequireAuth><SellPage /></RequireAuth>} />
          <Route path="/mine" element={<RequireAuth><MyListingsPage /></RequireAuth>} />
          <Route path="/account" element={<RequireAuth><AccountPage /></RequireAuth>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
