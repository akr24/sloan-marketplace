import { BrowserRouter, Navigate, NavLink, Route, Routes, useLocation } from 'react-router-dom'
import './App.css'
import { AuthProvider } from './auth/AuthContext'
import { useAuth } from './auth/context'
import Icon from './components/Icon'
import AccountPage from './pages/AccountPage'
import BrowsePage from './pages/BrowsePage'
import EditListingPage from './pages/EditListingPage'
import ListingPage from './pages/ListingPage'
import MyListingsPage from './pages/MyListingsPage'
import SellPage from './pages/SellPage'
import SignInPage from './pages/SignInPage'

// Phone-first: a floating tab bar within thumb reach, with Sell as the one
// bright button.
function TabBar() {
  return (
    <nav className="tabbar" aria-label="Main">
      <NavLink to="/" end aria-label="Browse">
        <Icon name="home" size={22} stroke={2} />
      </NavLink>
      <NavLink to="/mine" aria-label="My listings">
        <Icon name="ticket" size={22} stroke={2} />
      </NavLink>
      <NavLink to="/account" aria-label="Account">
        <Icon name="user" size={22} stroke={2} />
      </NavLink>
      <NavLink to="/sell" className="tab-sell">
        <Icon name="plus" size={18} stroke={2.6} />
        Sell
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
