import { useState, useEffect } from 'react'
import { getProviders, getSettings, getSubscriptionContent } from '../api'

export default function SubscriptionLinks() {
  const [providers, setProviders] = useState<any[]>([])
  const [copied, setCopied] = useState<string | null>(null)
  const [viewContent, setViewContent] = useState<string | null>(null)
  const [contentName, setContentName] = useState<string>('')
  const [contentData, setContentData] = useState<string>('')
  const [contentLoading, setContentLoading] = useState(false)
  const [contentTime, setContentTime] = useState<string>('')
  const [subBaseURL, setSubBaseURL] = useState('')

  useEffect(() => {
    getProviders().then(p => setProviders(Array.isArray(p) ? p : [])).catch(() => {})
    getSettings().then(s => { if (s?.sub_base_url) setSubBaseURL(s.sub_base_url) }).catch(() => {})
  }, [])

  const getSubURL = (clashName: string) => {
    if (subBaseURL) {
      const base = subBaseURL.replace(/\/+$/, '')
      return `${base}/sub/${clashName}`
    }
    return `${window.location.protocol}//${window.location.hostname}:9090/sub/${clashName}`
  }

  const copyURL = (url: string) => {
    try {
      navigator.clipboard.writeText(url)
    } catch {
      const ta = document.createElement('textarea')
      ta.value = url
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    setCopied(url)
    setTimeout(() => setCopied(null), 2000)
  }

  const openContent = async (clashName: string) => {
    setContentName(clashName)
    setContentLoading(true)
    setViewContent('loading')
    try {
      const res = await getSubscriptionContent(clashName)
      if (res.error) {
        setContentData('暂无缓存内容，请先刷新')
        setContentTime('')
      } else {
        setContentData(res.yaml)
        setContentTime(res.last_fetch || '')
      }
      setViewContent('loaded')
    } catch {
      setContentData('加载失败')
      setContentTime('')
      setViewContent('loaded')
    }
    setContentLoading(false)
  }

  const closeContent = () => {
    setViewContent(null)
    setContentData('')
    setContentTime('')
  }

  if (providers.length === 0) {
    return (
      <div>
        <h2 style={{ fontSize: 20, marginBottom: 16 }}>订阅链接</h2>
        <div className="empty-state">
          <p>暂无订阅链接</p>
          <p style={{ fontSize: 13 }}>添加 Provider 后会自动生成订阅链接</p>
        </div>
      </div>
    )
  }

  return (
    <div>
      <h2 style={{ fontSize: 20, marginBottom: 16 }}>订阅链接</h2>
      <div className="card">
        <table>
          <thead>
            <tr>
              <th>名称</th>
              <th>本地订阅地址</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {providers.map(p => {
              const url = getSubURL(p.clash_name)
              return (
                <tr key={p.id}>
                  <td><strong>{p.clash_name}</strong></td>
                  <td>
                    <code style={{ fontSize: 12, wordBreak: 'break-all' }}>{url}</code>
                  </td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    <button className="btn btn-sm btn-primary" onClick={() => copyURL(url)} style={{ marginRight: 8 }}>
                      {copied === url ? '已复制' : '复制'}
                    </button>
                    <button className="btn btn-sm" onClick={() => openContent(p.clash_name)}>
                      查看内容
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {viewContent && (
        <div className="modal-overlay" onClick={closeContent}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 900 }}>
            <div className="modal-title">
              {contentName} - 订阅内容
              <button className="btn btn-sm" onClick={closeContent} style={{ float: 'right' }}>✕</button>
            </div>
            {contentTime && <p style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>上次更新: {contentTime}</p>}
            <pre style={{
              background: '#1e1e1e',
              color: '#d4d4d4',
              padding: 16,
              borderRadius: 4,
              fontSize: 13,
              lineHeight: 1.5,
              overflow: 'auto',
              maxHeight: 500,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
            }}>{contentData}</pre>
            <div className="modal-actions">
              <button className="btn" onClick={closeContent}>关闭</button>
            </div>
          </div>
        </div>
      )}

      <div className="card" style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
        <strong>使用说明：</strong>将上述地址配置到 Clash 配置文件的 proxy-providers 中，如：
        <pre style={{ marginTop: 8, background: '#f5f5f5', padding: 12, borderRadius: 4, fontSize: 12 }}>
{`proxy-providers:
  my-provider:
    type: http
    url: "${providers.length > 0 ? getSubURL(providers[0].clash_name).replace(providers[0].clash_name, '你的订阅名称') : 'http://subme:9090/sub/你的订阅名称'}"
    interval: 3600`}
        </pre>
      </div>
    </div>
  )
}
