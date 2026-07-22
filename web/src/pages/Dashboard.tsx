import { useState, useEffect } from 'react'
import { getDashboard, refreshAll, refreshProvider } from '../api'

export default function Dashboard() {
  const [data, setData] = useState<any>(null)
  const [refreshing, setRefreshing] = useState(false)

  const load = async () => {
    const d = await getDashboard().catch(() => null)
    setData(d)
  }

  useEffect(() => { load() }, [])

  const handleRefreshAll = async () => {
    setRefreshing(true)
    await refreshAll()
    setTimeout(() => { setRefreshing(false); load() }, 2000)
  }

  const handleRefresh = async (id: number) => {
    await refreshProvider(id)
    setTimeout(load, 2000)
  }

  if (!data) return null

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ fontSize: 20 }}>仪表盘</h2>
        <button className="btn btn-primary" onClick={handleRefreshAll} disabled={refreshing}>
          {refreshing ? '刷新中...' : '刷新全部'}
        </button>
      </div>

      <div style={{ display: 'flex', gap: 16, marginBottom: 20 }}>
        <div className="card" style={{ flex: 1, textAlign: 'center', padding: 20 }}>
          <div style={{ fontSize: 28, fontWeight: 700 }}>{data.total}</div>
          <div style={{ fontSize: 13, color: '#888' }}>Provider</div>
        </div>
        <div className="card" style={{ flex: 1, textAlign: 'center', padding: 20 }}>
          <div style={{ fontSize: 28, fontWeight: 700 }}>{data.cached}</div>
          <div style={{ fontSize: 13, color: '#888' }}>已缓存</div>
        </div>
        <div className="card" style={{ flex: 1, textAlign: 'center', padding: 20 }}>
          <div style={{ fontSize: 28, fontWeight: 700 }}>{data.total_proxies}</div>
          <div style={{ fontSize: 13, color: '#888' }}>代理节点</div>
        </div>
      </div>

      {data.providers.length === 0 ? (
        <div className="empty-state">
          <p>还没有添加 Provider</p>
          <p style={{ fontSize: 13 }}>前往「Provider 管理」添加第一个订阅</p>
        </div>
      ) : (
        <div className="grid">
          {data.providers.map((p: any) => (
            <div className="card" key={p.id} style={{ position: 'relative' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <div style={{ fontWeight: 600, fontSize: 15 }}>{p.clash_name}</div>
                  <div style={{ fontSize: 12, color: '#888', marginTop: 4 }}>{p.collector}</div>
                </div>
                <span className={`badge ${p.has_cache ? 'badge-ok' : 'badge-warn'}`}>
                  {p.has_cache ? '已缓存' : '未缓存'}
                </span>
              </div>
              {p.panel_url && (
                <div style={{ fontSize: 12, color: '#999', marginTop: 8, wordBreak: 'break-all' }}>{p.panel_url}</div>
              )}
              <div style={{ display: 'flex', gap: 16, marginTop: 10, fontSize: 13 }}>
                <span>{p.proxy_count} 节点</span>
                <span>{p.last_fetch ? `更新: ${p.last_fetch}` : '暂无缓存'}</span>
              </div>
              <div style={{ marginTop: 10, display: 'flex', gap: 6 }}>
                <button className="btn btn-sm" onClick={() => handleRefresh(p.id)}>刷新</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
