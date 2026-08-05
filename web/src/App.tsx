import { BrowserRouter, Link, Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth/AuthContext'
import { GamePage } from './pages/GamePage'
import { LobbyPage } from './pages/LobbyPage'
import { LoginPage } from './pages/LoginPage'
import { ProfilePage } from './pages/ProfilePage'

function Shell() {
  const { profile, loading, logout } = useAuth()
  if (loading) return <div className="page-loading">loading…</div>
  if (!profile) return <Navigate to="/login" replace />
  return (
    <>
      <header className="topbar">
        <Link to="/" className="brand">
          ♞ chess arena
        </Link>
        <div className="topbar-right">
          <Link to={`/profile/${profile.id}`}>{profile.display_name ?? 'anonymous'}</Link>
          <button className="ghost" onClick={() => void logout()}>
            sign out
          </button>
        </div>
      </header>
      <main className="content">
        <Outlet />
      </main>
    </>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<Shell />}>
            <Route path="/" element={<LobbyPage />} />
            <Route path="/game/:id" element={<GamePage />} />
            <Route path="/profile/:id" element={<ProfilePage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
