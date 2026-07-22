import { useState, useEffect } from 'react'
import { healthCheck } from './api'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Providers from './pages/Providers'
import Collectors from './pages/Collectors'
import SubscriptionLinks from './pages/SubscriptionLinks'
import Logs from './pages/Logs'
import Settings from './pages/Settings'

type Page = 'dashboard' | 'providers' | 'collectors' | 'subscriptions' | 'logs' | 'settings'

export default function App() {
  const [page, setPage] = useState<Page>('dashboard')
  const [authenticated, setAuthenticated] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (token) {
      setAuthenticated(true)
    }
    setLoading(false)
  }, [])

  const handleLogin = () => {
    setAuthenticated(true)
    setPage('dashboard')
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    setAuthenticated(false)
    setPage('dashboard')
  }

  if (loading) return null

  if (!authenticated) {
    return <Login onLogin={handleLogin} />
  }

  const navLinks: { key: Page; label: string }[] = [
    { key: 'dashboard', label: '仪表盘' },
    { key: 'providers', label: 'Provider 管理' },
    { key: 'collectors', label: 'Collector 管理' },
    { key: 'subscriptions', label: '订阅链接' },
    { key: 'logs', label: '运行日志' },
    { key: 'settings', label: '系统设置' },
  ]

  return (
    <div>
      <nav className="nav">
        <div className="nav-brand">SubMe</div>
        {navLinks.map(l => (
          <a
            key={l.key}
            className={`nav-link ${page === l.key ? 'active' : ''}`}
            href="#"
            onClick={e => { e.preventDefault(); setPage(l.key) }}
          >
            {l.label}
          </a>
        ))}
        <div style={{ marginLeft: 'auto' }}>
          <a className="nav-link" href="#" onClick={e => { e.preventDefault(); handleLogout() }}>
            退出
          </a>
        </div>
      </nav>
      <div className="main">
        {page === 'dashboard' && <Dashboard />}
        {page === 'providers' && <Providers />}
        {page === 'collectors' && <Collectors />}
        {page === 'subscriptions' && <SubscriptionLinks />}
        {page === 'logs' && <Logs />}
        {page === 'settings' && <Settings />}
      </div>
    </div>
  )
}
