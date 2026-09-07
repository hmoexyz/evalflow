import type { EvaluationItem, Submission, WorkflowForm } from './types'

const TOKEN_KEY = 'evalflow_token'
const USERNAME_KEY = 'evalflow_username'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function getUsername() {
  return localStorage.getItem(USERNAME_KEY)
}

export function setAuth(token: string, username: string) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USERNAME_KEY, username)
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USERNAME_KEY)
}

interface ApiError {
  error?: string
}

async function req<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  if (options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(path, { ...options, headers })
  const data = (await res.json().catch(() => ({}))) as ApiError & T
  if (!res.ok) {
    // 除登录/注册接口外的 401 均视为会话失效：清除登录态并退回登录页
    if (res.status === 401 && !path.startsWith('/api/login') && !path.startsWith('/api/register')) {
      clearAuth()
      window.location.href = '/login'
      throw new Error('登录已过期，请重新登录')
    }
    const err = new Error(data.error || `请求失败 (${res.status})`) as Error & { status?: number }
    err.status = res.status
    throw err
  }
  return data
}

export const api = {
  register: (username: string, password: string) =>
    req<{ token: string; username: string }>('/api/register', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  login: (username: string, password: string) =>
    req<{ token: string; username: string }>('/api/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  changePassword: (old_password: string, new_password: string) =>
    req<{ ok: boolean }>('/api/password', {
      method: 'POST',
      body: JSON.stringify({ old_password, new_password }),
    }),

  listItems: () => req<EvaluationItem[]>('/api/items'),
  createItem: (name: string, description: string) =>
    req<EvaluationItem>('/api/items', { method: 'POST', body: JSON.stringify({ name, description }) }),
  updateItem: (id: number, name: string, description: string) =>
    req<EvaluationItem>(`/api/items/${id}`, { method: 'PUT', body: JSON.stringify({ name, description }) }),
  deleteItem: (id: number) => req<void>(`/api/items/${id}`, { method: 'DELETE' }),

  listForms: () => req<WorkflowForm[]>('/api/forms'),
  createForm: (name: string, item_ids: number[]) =>
    req<WorkflowForm>('/api/forms', { method: 'POST', body: JSON.stringify({ name, item_ids }) }),
  updateForm: (id: number, name: string, item_ids: number[]) =>
    req<WorkflowForm>(`/api/forms/${id}`, { method: 'PUT', body: JSON.stringify({ name, item_ids }) }),
  deleteForm: (id: number) => req<void>(`/api/forms/${id}`, { method: 'DELETE' }),
  publishForm: (id: number) =>
    req<{ token: string; url: string }>(`/api/forms/${id}/publish`, { method: 'POST' }),
  unpublishForm: (id: number) => req<void>(`/api/forms/${id}/unpublish`, { method: 'POST' }),
  listSubmissions: (id: number) => req<Submission[]>(`/api/forms/${id}/submissions`),
  listAllSubmissions: () => req<Submission[]>('/api/submissions'),

  getSharedForm: (token: string) => req<WorkflowForm>(`/api/share/${token}`),
  listPublishedForms: () => req<WorkflowForm[]>('/api/share'),
  submit: (
    token: string,
    restaurant: string,
    evaluator: string,
    scores: { item_id: number; score: number; evidence: string[] }[],
  ) =>
    req<{ submission: Submission }>(`/api/share/${token}/submissions`, {
      method: 'POST',
      body: JSON.stringify({ restaurant, evaluator, scores }),
    }),
  getResult: (token: string) =>
    req<{ form: WorkflowForm; submission: Submission }>(`/api/results/${token}`),

  upload: async (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    const res = await fetch('/api/upload', { method: 'POST', body: fd })
    const data = (await res.json().catch(() => ({}))) as ApiError & { url: string }
    if (!res.ok) throw new Error(data.error || '上传失败')
    return data
  },
}
