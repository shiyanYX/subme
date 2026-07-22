import { useState, useEffect, useRef } from 'react'
import { getCollectors, uploadCollector } from '../api'

export default function Collectors() {
  const [collectors, setCollectors] = useState<string[]>([])
  const [name, setName] = useState('')
  const [jsFile, setJsFile] = useState<File | null>(null)
  const [yamlFile, setYamlFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [msg, setMsg] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)
  const yamlRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    const c = await getCollectors().catch(() => [])
    setCollectors(Array.isArray(c) ? c : [])
  }

  useEffect(() => { load() }, [])

  const handleUpload = async () => {
    if (!name || !jsFile) return
    setUploading(true)
    setMsg('')
    try {
      const res = await uploadCollector(name, jsFile, yamlFile || undefined)
      if (res.error) {
        setMsg('上传失败: ' + res.error)
      } else {
        setMsg('上传成功: ' + name)
        setName('')
        setJsFile(null)
        setYamlFile(null)
        if (fileRef.current) fileRef.current.value = ''
        if (yamlRef.current) yamlRef.current.value = ''
        load()
      }
    } catch (e: any) {
      setMsg('上传失败: ' + (e.message || 'unknown'))
    }
    setUploading(false)
  }

  return (
    <div>
      <h2 style={{ fontSize: 20, marginBottom: 16 }}>Collector 管理</h2>

      <div className="card" style={{ marginBottom: 20 }}>
        <div className="card-title">上传 Collector</div>
        <div className="form-group">
          <label>名称（字母、数字、横线、下划线、点）</label>
          <input value={name} onChange={e => setName(e.target.value)} placeholder="my-provider" />
        </div>
        <div className="form-group">
          <label>collector.js</label>
          <input type="file" ref={fileRef} accept=".js" onChange={e => setJsFile(e.target.files?.[0] || null)} />
        </div>
        <div className="form-group">
          <label>config.yaml <span style={{ color: '#999' }}>（可选，不传则自动生成模板）</span></label>
          <input type="file" ref={yamlRef} accept=".yaml,.yml" onChange={e => setYamlFile(e.target.files?.[0] || null)} />
        </div>
        {msg && <div style={{ marginBottom: 10, fontSize: 13, color: msg.includes('成功') ? 'green' : 'red' }}>{msg}</div>}
        <button className="btn btn-primary" onClick={handleUpload} disabled={uploading || !name || !jsFile}>
          {uploading ? '上传中...' : '上传'}
        </button>
      </div>

      <div className="card">
        <div className="card-title">已安装的 Collector</div>
        {collectors.length === 0 ? (
          <div className="empty-state"><p>暂无 Collector</p></div>
        ) : (
          <table>
            <thead>
              <tr><th>名称</th></tr>
            </thead>
            <tbody>
              {collectors.map(c => (
                <tr key={c}><td><strong>{c}</strong></td></tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
