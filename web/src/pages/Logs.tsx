import { useState, useEffect, useRef } from 'react'

export default function Logs() {
  const [logs, setLogs] = useState<any[]>([])
  const [connected, setConnected] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let eventSource: EventSource | null = null
    let reconnectTimer: ReturnType<typeof setTimeout>

    const connect = () => {
      eventSource = new EventSource('/api/logs')

      eventSource.onopen = () => setConnected(true)

      eventSource.onmessage = (e) => {
        try {
          const entry = JSON.parse(e.data)
          setLogs(prev => {
            const next = [...prev, entry]
            return next.length > 500 ? next.slice(next.length - 500) : next
          })
        } catch {}
      }

      eventSource.onerror = () => {
        setConnected(false)
        eventSource?.close()
        reconnectTimer = setTimeout(connect, 3000)
      }
    }

    connect()

    return () => {
      eventSource?.close()
      clearTimeout(reconnectTimer)
    }
  }, [])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ fontSize: 20 }}>运行日志</h2>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <span className={`badge ${connected ? 'badge-ok' : 'badge-fail'}`}>
            {connected ? '已连接' : '已断开'}
          </span>
          <button className="btn btn-sm" onClick={() => setLogs([])}>清空</button>
        </div>
      </div>

      <div className="log-container">
        {logs.length === 0 ? (
          <div style={{ color: '#888' }}>等待日志...</div>
        ) : (
          logs.map((log, i) => (
            <div className="log-entry" key={i}>
              <span className="log-time">{new Date(log.time).toLocaleTimeString()}</span>
              {' '}
              <span className={`log-${log.level}`}>[{log.level.toUpperCase()}]</span>
              {' '}
              <span>{log.message}</span>
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
