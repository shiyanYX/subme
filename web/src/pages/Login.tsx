import { useState, useEffect } from 'react'
import { login, register, healthCheck } from '../api'

export default function Login({ onLogin }: { onLogin: () => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [serverOk, setServerOk] = useState(false)

  useEffect(() => {
    healthCheck().then(setServerOk)
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      if (mode === 'register') {
        await register(username, password)
        setMode('login')
        setError('注册成功，请登录')
        return
      }
      const data = await login(username, password)
      if (data.token) {
        onLogin()
      } else {
        setError('登录失败')
      }
    } catch (err: any) {
      if (err.message === 'unauthorized') {
        setError('用户名或密码错误')
      } else {
        const text = await err.text?.() || err.message
        setError(typeof text === 'string' ? text : '请求失败')
      }
    }
  }

  const tryRegister = async () => {
    setError('')
    try {
      await register(username, password)
      setMode('login')
      setError('注册成功，请登录')
    } catch (err: any) {
      const text = await err.text?.()
      setError(typeof text === 'string' ? text : '注册失败，管理员可能已存在')
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>SubMe</h1>
        {!serverOk && <div className="alert alert-error">无法连接到服务器</div>}
        {error && <div className={`alert ${error.includes('成功') ? 'alert-success' : 'alert-error'}`}>{error}</div>}
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>用户名</label>
            <input value={username} onChange={e => setUsername(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>密码</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} required />
          </div>
          {mode === 'login' ? (
            <>
              <button type="submit" className="btn btn-primary" style={{ width: '100%', justifyContent: 'center' }}>
                登录
              </button>
              <div style={{ textAlign: 'center', marginTop: 12, fontSize: 14 }}>
                <a href="#" onClick={e => { e.preventDefault(); setMode('register') }} style={{ color: 'var(--primary)' }}>
                  注册管理员账号
                </a>
              </div>
            </>
          ) : (
            <>
              <button type="button" className="btn btn-primary" style={{ width: '100%', justifyContent: 'center' }} onClick={tryRegister}>
                注册
              </button>
              <div style={{ textAlign: 'center', marginTop: 12, fontSize: 14 }}>
                <a href="#" onClick={e => { e.preventDefault(); setMode('login') }} style={{ color: 'var(--primary)' }}>
                  返回登录
                </a>
              </div>
            </>
          )}
        </form>
      </div>
    </div>
  )
}
