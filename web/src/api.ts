const BASE = ''

async function request(path: string, options?: RequestInit) {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string>),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${BASE}${path}`, { ...options, headers })
  if (res.status === 401) {
    localStorage.removeItem('token')
    window.location.hash = '#/login'
    throw new Error('unauthorized')
  }
  return res
}

export async function register(username: string, password: string) {
  const res = await request('/api/register', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  return res.json()
}

export async function login(username: string, password: string) {
  const res = await request('/api/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  const data = await res.json()
  if (data.token) localStorage.setItem('token', data.token)
  return data
}

export async function getProviders() {
  const res = await request('/api/providers')
  return res.json()
}

export async function getProvider(id: number) {
  const res = await request(`/api/providers/${id}`)
  return res.json()
}

export async function createProvider(p: any) {
  const res = await request('/api/providers', {
    method: 'POST',
    body: JSON.stringify(p),
  })
  return res.json()
}

export async function updateProvider(id: number, p: any) {
  const res = await request(`/api/providers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(p),
  })
  return res.json()
}

export async function deleteProvider(id: number) {
  await request(`/api/providers/${id}`, { method: 'DELETE' })
}

export async function testProvider(config: any) {
  const res = await request('/api/providers/0/test', {
    method: 'POST',
    body: JSON.stringify(config),
  })
  return res.json()
}

export async function refreshProvider(id: number) {
  await request(`/api/providers/${id}/refresh`, { method: 'POST' })
}

export async function refreshAll() {
  await request('/api/refresh', { method: 'POST' })
}

export async function getSettings() {
  const res = await request('/api/settings')
  return res.json()
}

export async function updateSettings(s: any) {
  const res = await request('/api/settings', {
    method: 'PUT',
    body: JSON.stringify(s),
  })
  return res.json()
}

export async function getCollectors() {
  const res = await request('/api/configs')
  return res.json()
}

export async function getSubscriptionContent(clashName: string) {
  const res = await request(`/api/sub/${clashName}/content`)
  return res.json()
}

export async function healthCheck() {
  try {
    const res = await fetch(`${BASE}/api/health`)
    return res.ok
  } catch { return false }
}

export async function getDashboard() {
  const res = await request('/api/dashboard')
  return res.json()
}

export async function uploadCollector(name: string, collectorFile: File, configFile?: File) {
  const form = new FormData()
  form.append('name', name)
  form.append('collector', collectorFile)
  if (configFile) form.append('config', configFile)
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`
  const res = await fetch(`${BASE}/api/configs/upload`, {
    method: 'POST',
    headers,
    body: form,
  })
  return res.json()
}
