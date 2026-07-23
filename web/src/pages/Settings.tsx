import { useState, useEffect } from 'react'
import { getSettings, updateSettings } from '../api'

export default function Settings() {
  const [settings, setSettings] = useState({
    port: 9090,
    refresh_interval: 3600,
    proxy: '',
    sub_base_url: '',
    wxpusher: { app_token: '', uids: [] as string[] },
    notify_on: { collect_failure: true, refresh_failure: true },
  })
  const [saved, setSaved] = useState(false)
  const [uidInput, setUidInput] = useState('')

  useEffect(() => {
    getSettings().then(s => {
      if (s && s.port) setSettings(s)
    }).catch(() => {})
  }, [])

  const handleSave = async () => {
    try {
      await updateSettings({
        ...settings,
        wxpusher: {
          ...settings.wxpusher,
          uids: typeof settings.wxpusher.uids === 'string'
            ? (settings.wxpusher.uids as any).split(',').filter(Boolean)
            : settings.wxpusher.uids,
        }
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {}
  }

  const addUID = () => {
    if (!uidInput.trim()) return
    setSettings({
      ...settings,
      wxpusher: { ...settings.wxpusher, uids: [...settings.wxpusher.uids, uidInput.trim()] }
    })
    setUidInput('')
  }

  const removeUID = (idx: number) => {
    setSettings({
      ...settings,
      wxpusher: { ...settings.wxpusher, uids: settings.wxpusher.uids.filter((_: any, i: number) => i !== idx) }
    })
  }

  return (
    <div>
      <h2 style={{ fontSize: 20, marginBottom: 16 }}>系统设置</h2>

      {saved && <div className="alert alert-success">设置已保存</div>}

      <div className="card">
        <div className="card-title">基本设置</div>
        <div className="form-row">
          <div className="form-group">
            <label>HTTP 端口</label>
            <input type="number" value={settings.port} onChange={e => setSettings({...settings, port: parseInt(e.target.value) || 9090})} />
          </div>
          <div className="form-group">
            <label>默认刷新间隔（秒）</label>
            <input type="number" value={settings.refresh_interval} onChange={e => setSettings({...settings, refresh_interval: parseInt(e.target.value) || 3600})} />
          </div>
        </div>
        <div className="form-group">
          <label>全局代理</label>
          <input value={settings.proxy} onChange={e => setSettings({...settings, proxy: e.target.value})} placeholder="socks5://127.0.0.1:7890 或留空" />
        </div>
        <div className="form-group">
          <label>订阅地址前缀</label>
          <input value={settings.sub_base_url} onChange={e => setSettings({...settings, sub_base_url: e.target.value})} placeholder="留空则自动生成（如 http://192.168.1.83:9090）" />
          <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>设置后订阅链接显示为：{settings.sub_base_url || '{自动}'}/sub/&#123;名称&#125;</span>
        </div>
      </div>

      <div className="card">
        <div className="card-title">WxPusher 通知</div>
        <div className="form-group">
          <label>App Token</label>
          <input value={settings.wxpusher.app_token} onChange={e => setSettings({...settings, wxpusher: {...settings.wxpusher, app_token: e.target.value}})} placeholder="AT_xxx" />
        </div>
        <div className="form-group">
          <label>UID 列表</label>
          {settings.wxpusher.uids.map((uid: string, i: number) => (
            <div key={i} style={{ display: 'flex', gap: 8, marginBottom: 4, alignItems: 'center' }}>
              <code style={{ flex: 1, padding: '4px 8px', background: '#f5f5f5', borderRadius: 4 }}>{uid}</code>
              <button className="btn btn-sm btn-danger" onClick={() => removeUID(i)}>×</button>
            </div>
          ))}
          <div style={{ display: 'flex', gap: 8 }}>
            <input value={uidInput} onChange={e => setUidInput(e.target.value)} placeholder="UID_xxx" style={{ flex: 1 }} />
            <button className="btn btn-sm" onClick={addUID}>添加</button>
          </div>
        </div>
        <div className="form-group">
          <label>通知条件</label>
          <div className="checkbox-group">
            <label>
              <input type="checkbox" checked={settings.notify_on.collect_failure}
                onChange={e => setSettings({...settings, notify_on: {...settings.notify_on, collect_failure: e.target.checked}})} />
              订阅链接获取失败
            </label>
            <label>
              <input type="checkbox" checked={settings.notify_on.refresh_failure}
                onChange={e => setSettings({...settings, notify_on: {...settings.notify_on, refresh_failure: e.target.checked}})} />
              刷新失败
            </label>
          </div>
        </div>
      </div>

      <button className="btn btn-primary" onClick={handleSave}>保存设置</button>
    </div>
  )
}
