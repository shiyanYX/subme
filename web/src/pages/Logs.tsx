import { useState, useEffect, useRef, useMemo } from 'react'

const LEVELS = ['all', 'debug', 'info', 'warn', 'error'] as const
type LogLevel = typeof LEVELS[number]

const LEVEL_ORDER: Record<string, number> = { debug: 0, info: 1, warn: 2, error: 3 }

function levelMatch(filter: LogLevel, level: string): boolean {
  if (filter === 'all') return true
  return LEVEL_ORDER[level] >= LEVEL_ORDER[filter]
}

export default function Logs() {
  const [logs, setLogs] = useState<any[]>([])
  const [connected, setConnected] = useState(false)
  const [levelFilter, setLevelFilter] = useState<LogLevel>('all')
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

  const filteredLogs = useMemo(() => {
    return levelFilter === 'all' ? logs : logs.filter(l => levelMatch(levelFilter, l.level))
  }, [logs, levelFilter])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [filteredLogs])

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ fontSize: 20 }}>运行日志</h2>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <select
            className="select"
            value={levelFilter}
            onChange={e => setLevelFilter(e.target.value as LogLevel)}
            style={{ fontSize: 13, padding: '4px 8px' }}
          >
            {LEVELS.map(l => (
              <option key={l} value={l}>{l.toUpperCase()}</option>
            ))}
          </select>
          <span className={`badge ${connected ? 'badge-ok' : 'badge-fail'}`}>
            {connected ? '已连接' : '已断开'}
          </span>
          <button className="btn btn-sm" onClick={() => setLogs([])}>清空</button>
        </div>
      </div>

      <div className="log-container">
        {filteredLogs.length === 0 ? (
          <div style={{ color: '#888' }}>{logs.length === 0 ? '等待日志...' : '无匹配日志'}</div>
        ) : (
          filteredLogs.map((log, i) => (
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
