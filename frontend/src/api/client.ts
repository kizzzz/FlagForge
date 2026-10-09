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
    const payload = response.status === 204 ? undefined : await response.json().catch(error => {
      if (error instanceof DOMException && error.name === 'AbortError') throw error
      return undefined
    })
    if (options.signal?.aborted) throw new DOMException('请求已取消', 'AbortError')
    if (!response.ok) {
      const error = payload?.error
      throw new ApiError(error?.code ?? 'INTERNAL_ERROR', error?.message ?? `请求失败（HTTP ${response.status}），请重试`, error?.details ?? {}, response.status)
    }
    return payload as T
  } catch (error) {
    if (options.signal?.aborted || (error instanceof DOMException && error.name === 'AbortError')) throw error
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

export type RuleInput = Pick<Rule, 'attribute' | 'operator' | 'expected_value' | 'roll_value' | 'priority'>
export type Cause = 'matched' | 'attribute_missing' | 'value_not_equal' | 'value_not_in_list' | 'type_mismatch' | 'short_circuited_by_disabled_flag'
export interface TraceStep { rule_id?: string; priority?: number; attribute?: string; operator?: Rule['operator']; matched: boolean; cause: Cause }
export interface Reason { type: 'rule_hit' | 'default' | 'disabled'; rule_id?: string; priority?: number; summary?: string }
export interface Evaluation { flag_key: string; environment: Environment; enabled: boolean; value: boolean; reason: Reason; trace: TraceStep[]; evaluated_at: string }
export interface EvaluationInput { flag_key: string; environment: Environment; user_attributes: Record<string, unknown> }
export interface BatchResult {
  results: { index: number; value: boolean; reason: Reason; trace: TraceStep[] }[]
  summary: { total: number; true_count: number; false_count: number; default_count: number; hit_rules: { rule_id: string; priority: number; attribute: string; count: number }[] }
}
export const ruleApi = {
  create: (id: string, body: RuleInput, options?: RequestOptions) => request<Rule>(`/flags/${encodeURIComponent(id)}/rules`, { method: 'POST', body: JSON.stringify(body) }, options),
  update: (id: string, rid: string, body: RuleInput, options?: RequestOptions) => request<Rule>(`/flags/${encodeURIComponent(id)}/rules/${encodeURIComponent(rid)}`, { method: 'PATCH', body: JSON.stringify(body) }, options),
  remove: (id: string, rid: string, options?: RequestOptions) => request<void>(`/flags/${encodeURIComponent(id)}/rules/${encodeURIComponent(rid)}`, { method: 'DELETE' }, options),
}
export const evaluationApi = {
  evaluate: (body: EvaluationInput, options?: RequestOptions) => request<Evaluation>('/evaluate', { method: 'POST', body: JSON.stringify(body) }, options),
  batch: (body: Omit<EvaluationInput, 'user_attributes'> & { user_attributes_list: Record<string, unknown>[] }, options?: RequestOptions) => request<BatchResult>('/evaluate/batch', { method: 'POST', body: JSON.stringify(body) }, options),
}
export const copyFlag = (id: string, target_environment: Environment, options?: RequestOptions) => request<Flag>(`/flags/${encodeURIComponent(id)}/copy`, { method: 'POST', body: JSON.stringify({ target_environment }) }, options)
