import { useState, useEffect, useRef } from 'react'
import { getCollectors, uploadCollector, deleteCollector, renameCollector } from '../api'

export default function Collectors() {
  const [collectors, setCollectors] = useState<string[]>([])
  const [name, setName] = useState('')
  const [jsFile, setJsFile] = useState<File | null>(null)
  const [yamlFile, setYamlFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [msg, setMsg] = useState('')
  const [renaming, setRenaming] = useState<string | null>(null)
  const [newName, setNewName] = useState('')
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
      const exists = collectors.includes(name)
      const res = await uploadCollector(name, jsFile, yamlFile || undefined)
      if (res.error) {
        setMsg('上传失败: ' + res.error)
      } else {
        setMsg(exists ? '覆盖成功: ' + name : '上传成功: ' + name)
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

  const handleDelete = async (c: string) => {
    if (!confirm(`确定删除 collector「${c}」？`)) return
    const res = await deleteCollector(c)
    if (res.error) {
      setMsg('删除失败: ' + res.error)
    } else {
      setMsg('删除成功: ' + c)
      load()
    }
  }

  const handleRename = async (oldName: string) => {
    if (!newName.trim() || newName.trim() === oldName) {
      setRenaming(null)
      setNewName('')
      return
    }
    const res = await renameCollector(oldName, newName.trim())
    if (res.error) {
      setMsg('重命名失败: ' + res.error)
    } else {
      setMsg('重命名成功: ' + oldName + ' -> ' + newName.trim())
      setRenaming(null)
      setNewName('')
      load()
    }
  }

  return (
    <div>
      <h2 style={{ fontSize: 20, marginBottom: 16 }}>Collector 管理</h2>

      <div className="card" style={{ marginBottom: 20 }}>
        <div className="card-title">上传 / 覆盖 Collector</div>
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
        {msg && <div style={{ marginBottom: 10, fontSize: 13, color: msg.includes('失败') ? 'red' : 'green' }}>{msg}</div>}
        <button className="btn btn-primary" onClick={handleUpload} disabled={uploading || !name || !jsFile}>
          {uploading ? '上传中...' : (collectors.includes(name) ? '覆盖上传' : '上传')}
        </button>
      </div>

      <div className="card">
        <div className="card-title">已安装的 Collector</div>
        {collectors.length === 0 ? (
          <div className="empty-state"><p>暂无 Collector</p></div>
        ) : (
          <table>
            <thead>
              <tr><th>名称</th><th>操作</th></tr>
            </thead>
            <tbody>
              {collectors.map(c => (
                <tr key={c}>
                  <td>
                    {renaming === c ? (
                      <input value={newName} onChange={e => setNewName(e.target.value)}
                        onKeyDown={e => { if (e.key === 'Enter') handleRename(c); if (e.key === 'Escape') { setRenaming(null); setNewName('') } }}
                        autoFocus style={{ width: 200 }} placeholder="输入新名称" />
                    ) : (
                      <strong>{c}</strong>
                    )}
                  </td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    {renaming === c ? (
                      <>
                        <button className="btn btn-sm btn-primary" onClick={() => handleRename(c)} style={{ marginRight: 8 }}>确定</button>
                        <button className="btn btn-sm" onClick={() => { setRenaming(null); setNewName('') }}>取消</button>
                      </>
                    ) : (
                      <>
                        <button className="btn btn-sm" onClick={() => { setRenaming(c); setNewName('') }} style={{ marginRight: 8 }}>重命名</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(c)}>删除</button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}