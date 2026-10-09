export const environments = ['development', 'staging', 'production'] as const
export type Environment = (typeof environments)[number]
export interface Flag {
  id: string
  name: string
  key: string
  environment: Environment
  enabled: boolean
  default_value: boolean
  description: string
  created_at: string
  updated_at: string
}
export interface Rule {
  id: string
  flag_id: string
  attribute: string
  operator: 'equals' | 'in'
  expected_value: string | number | boolean | (string | number | boolean)[]
  roll_value: boolean
  priority: number
  created_at: string
  updated_at: string
}
export interface FlagListItem extends Flag { rule_count: number }
export interface FlagDetail extends Flag { rules: Rule[] }
export interface History {
  id: number
  action: string
  operator: string
  summary: Record<string, unknown>
  created_at: string
}
export interface Paged<T> { items: T[]; page: number; page_size: number; total: number }
export type CreateFlag = Pick<Flag, 'name' | 'key' | 'environment' | 'default_value'> & Partial<Pick<Flag, 'description' | 'enabled'>>
export type UpdateFlag = Partial<Pick<Flag, 'name' | 'description' | 'default_value'>>
export interface ListFlagsQuery {
  environment?: Environment
  enabled?: boolean
  q?: string
  page?: number
  page_size?: number
}
export class ApiError extends Error {
  constructor(public code: string, message: string, public details: unknown = {}, public status = 0) {
    super(message)
    this.name = 'ApiError'
  }
}
type RequestOptions = { signal?: AbortSignal; inlineErrors?: string[] }
let reportError: (error: ApiError) => void = () => undefined
export function setApiErrorReporter(reporter: (error: ApiError) => void) {
  reportError = reporter
}
async function request<T>(path: string, init: RequestInit = {}, options: RequestOptions = {}): Promise<T> {
  try {
    const response = await fetch(`/api/v1${path}`, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init.headers },
      signal: options.signal,
    })
    const payload = response.status === 204 ? undefined : await response.json().catch(() => undefined)
    if (!response.ok) {
      const error = payload?.error
      throw new ApiError(error?.code ?? 'INTERNAL_ERROR', error?.message ?? `请求失败（HTTP ${response.status}），请重试`, error?.details ?? {}, response.status)
    }
    return payload as T
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    const apiError = error instanceof ApiError ? error : new ApiError('NETWORK_ERROR', '无法连接服务，请检查网络后重试')
    if (!options.inlineErrors?.includes(apiError.code)) reportError(apiError)
    throw apiError
  }
}
function queryString(query: object) {
  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== '') params.set(key, String(value))
  })
  return params.toString()
}
export const flagApi = {
  list: (query: ListFlagsQuery = {}, options?: RequestOptions) => request<Paged<FlagListItem>>(`/flags?${queryString(query)}`, {}, options),
  get: (id: string, options?: RequestOptions) => request<FlagDetail>(`/flags/${encodeURIComponent(id)}`, {}, options),
  create: (body: CreateFlag, options?: RequestOptions) => request<Flag>('/flags', { method: 'POST', body: JSON.stringify(body) }, options),
  update: (id: string, body: UpdateFlag, options?: RequestOptions) => request<Flag>(`/flags/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(body) }, options),
  setStatus: (id: string, enabled: boolean, options?: RequestOptions) => request<Flag>(`/flags/${encodeURIComponent(id)}/status`, { method: 'PATCH', body: JSON.stringify({ enabled }) }, options),
  history: (id: string, query: { page?: number; page_size?: number } = {}, options?: RequestOptions) => request<Paged<History>>(`/flags/${encodeURIComponent(id)}/history?${queryString(query)}`, {}, options),
}
