import { BrowserRouter, Navigate, NavLink, Route, Routes, useLocation } from 'react-router-dom'
import './App.css'
import { AuthProvider } from './auth/AuthContext'
import { useAuth } from './auth/context'
import AccountPage from './pages/AccountPage'
import BrowsePage from './pages/BrowsePage'
import EditListingPage from './pages/EditListingPage'
import ListingPage from './pages/ListingPage'
import ListingLivePage from './pages/ListingLivePage'
import NewListingPage from './pages/NewListingPage'
import SellPage from './pages/SellPage'
import SignInPage from './pages/SignInPage'

// Phone-first: a bottom tab bar within thumb reach, like a native app.
// Icons are simple strokes so they read as part of the wireframe style.
const icon = (d: string) => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d={d} />
  </svg>
)

function TabBar() {
  return (
    <nav className="tabbar" aria-label="Main">
      <NavLink to="/" end>
        {icon('M4 7a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v2a2 2 0 0 0 0 4v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-4a2 2 0 0 0 0-4zM14 5v14')}
        Browse
      </NavLink>
      <NavLink to="/sell">
        {icon('M12 5v14M5 12h14')}
        Sell
      </NavLink>
      <NavLink to="/account">
        {icon('M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM4 20c1.5-3.5 4.5-5 8-5s6.5 1.5 8 5')}
        Account
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
          <Route path="/listings/:id/live" element={<RequireAuth><ListingLivePage /></RequireAuth>} />
          <Route path="/sell" element={<RequireAuth><SellPage /></RequireAuth>} />
          <Route path="/sell/new" element={<RequireAuth><NewListingPage /></RequireAuth>} />
          {/* My listings now lives on the Sell tab; keep old links working. */}
          <Route path="/mine" element={<Navigate to="/sell" replace />} />
          <Route path="/account" element={<RequireAuth><AccountPage /></RequireAuth>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
