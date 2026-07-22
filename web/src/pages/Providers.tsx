import { useState, useEffect } from 'react'
import { getProviders, getProvider, createProvider, updateProvider, deleteProvider, testProvider, getCollectors, refreshProvider } from '../api'

export default function Providers() {
  const [providers, setProviders] = useState<any[]>([])
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<any>(null)
  const [testResult, setTestResult] = useState<any>(null)
  const [testing, setTesting] = useState(false)
  const [collectors, setCollectors] = useState<string[]>([])

  const load = async () => {
    const [p, c] = await Promise.all([
      getProviders().catch(() => []),
      getCollectors().catch(() => [])
    ])
    setProviders(Array.isArray(p) ? p : [])
    setCollectors(Array.isArray(c) ? c : [])
  }

  useEffect(() => { load() }, [])

  const formDefaults = {
    collector_name: '',
    clash_name: '',
    panel_url: '',
    landing_page: '',
    username: '',
    password: '',
    interval: 3600,
  }

  const [form, setForm] = useState<any>(formDefaults)

  const openAdd = () => {
    setForm(formDefaults)
    setEditing(null)
    setTestResult(null)
    setShowModal(true)
  }

  const openEdit = async (p: any) => {
    let pw = p.password || ''
    try {
      const detail = await getProvider(p.id)
      if (detail && detail.password) pw = detail.password
    } catch {}
    setForm({
      collector_name: p.collector_name || p.clash_name,
      clash_name: p.clash_name,
      panel_url: p.panel_url || '',
      landing_page: p.landing_page || '',
      username: p.username || '',
      password: pw,
      interval: p.interval || 3600,
    })
    setEditing(p)
    setTestResult(null)
    setShowModal(true)
  }

  const handleSave = async () => {
    if (editing) {
      await updateProvider(editing.id, form)
    } else {
      await createProvider({...form, collector_name: form.collector_name})
    }
    setShowModal(false)
    load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除该 Provider？')) return
    await deleteProvider(id)
    load()
  }

  const handleTest = async () => {
    if (!form.collector_name) { setTestResult({success: false, error: '请先选择 Collector'}); return }
    setTesting(true)
    setTestResult(null)
    const result = await testProvider({
      collector_name: form.collector_name,
      panel_url: form.panel_url,
      landing_page: form.landing_page,
      username: form.username,
      password: form.password,
    })
    setTestResult(result)
    setTesting(false)
  }

  const handleRefresh = async (id: number) => {
    await refreshProvider(id)
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ fontSize: 20 }}>Provider 管理</h2>
        <button className="btn btn-primary" onClick={openAdd}>添加 Provider</button>
      </div>

      {providers.length === 0 ? (
        <div className="empty-state">
          <p>暂无 Provider</p>
          <p style={{ fontSize: 13 }}>点击「添加 Provider」开始配置</p>
        </div>
      ) : (
        <div className="card">
          <table>
            <thead>
              <tr>
                <th>名称</th>
                <th>面板地址</th>
                <th>用户名</th>
                <th>刷新间隔</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {providers.map(p => (
                <tr key={p.id}>
                  <td><strong>{p.clash_name}</strong></td>
                  <td style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {p.panel_url || '-'}
                  </td>
                  <td>{p.username || '-'}</td>
                  <td>{p.interval ? `${p.interval / 60}分钟` : '默认'}</td>
                  <td>
                    <button className="btn btn-sm" onClick={() => openEdit(p)} style={{ marginRight: 4 }}>编辑</button>
                    <button className="btn btn-sm" onClick={() => handleRefresh(p.id)} style={{ marginRight: 4 }}>刷新</button>
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(p.id)}>删除</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-title">{editing ? '编辑 Provider' : '添加 Provider'}</div>

            {testResult && (
              <div className={`alert ${testResult.success ? 'alert-success' : 'alert-error'}`}>
                {testResult.success ? '连接成功！' : `连接失败: ${testResult.error || '未知错误'}`}
              </div>
            )}

            {!editing && (
              <div className="form-group">
                <label>Collector *</label>
                <select value={form.collector_name} onChange={e => setForm({...form, collector_name: e.target.value})}>
                  <option value="">-- 请选择 Collector --</option>
                  {collectors.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
                <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>
                  需要先在 collectors/ 目录下创建 collector.js
                </div>
              </div>
            )}

            <div className="form-row">
              <div className="form-group">
                <label>Clash 名称 *</label>
                <input value={form.clash_name} onChange={e => setForm({...form, clash_name: e.target.value})} placeholder="fieniao-jichang" disabled={!!editing} />
              </div>
              <div className="form-group">
                <label>刷新间隔（秒）</label>
                <input type="number" value={form.interval} onChange={e => setForm({...form, interval: parseInt(e.target.value) || 3600})} />
              </div>
            </div>

            <div className="form-group">
              <label>面板地址（panel_url）</label>
              <input value={form.panel_url} onChange={e => setForm({...form, panel_url: e.target.value})} placeholder="https://xxx.xyz" />
            </div>

            <div className="form-group">
              <label>发布页（landing_page）</label>
              <input value={form.landing_page} onChange={e => setForm({...form, landing_page: e.target.value})} placeholder="https://xxx.blogspot.com" />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>用户名</label>
                <input value={form.username} onChange={e => setForm({...form, username: e.target.value})} />
              </div>
              <div className="form-group">
                <label>密码</label>
                <input type="password" value={form.password} onChange={e => setForm({...form, password: e.target.value})} placeholder={editing ? '留空不修改' : ''} />
              </div>
            </div>

            <div className="modal-actions">
              <button className="btn" onClick={handleTest} disabled={testing}>
                {testing ? '测试中...' : '测试连接'}
              </button>
              <button className="btn" onClick={() => setShowModal(false)}>取消</button>
              <button className="btn btn-primary" onClick={handleSave}>
                {editing ? '保存' : '添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
