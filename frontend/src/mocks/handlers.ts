import { delay, http, HttpResponse } from 'msw'
import { environments } from '../api/client'
import type { Flag } from '../api/client'
import { flags, histories, recordHistory, rules } from './data'

const base = '/api/v1/flags'
const error = (status: number, code: string, message: string, details: unknown = {}) => HttpResponse.json({ error: { code, message, details } }, { status })
const notFound = () => error(404, 'FLAG_NOT_FOUND', 'Flag 不存在，请返回列表重试')
const invalid = (field: string, reason: string) => error(400, 'VALIDATION_ERROR', '请检查输入字段', [{ field, reason }])
function pagination(url: URL) {
  const page = Number(url.searchParams.get('page') ?? 1)
  const size = Number(url.searchParams.get('page_size') ?? 20)
  if (!Number.isInteger(page) || !Number.isInteger(size) || page < 1 || size < 0) return undefined
  return { page, page_size: size === 0 ? 20 : Math.min(size, 100) }
}
function paged<T>(items: T[], paging: { page: number; page_size: number }) {
  const { page, page_size } = paging
  return { items: items.slice((page - 1) * page_size, page * page_size), page, page_size, total: items.length }
}
async function body(request: Request): Promise<Record<string, unknown> | undefined> {
  try {
    const value = await request.json()
    return value && typeof value === 'object' && !Array.isArray(value) ? value : undefined
  } catch { return undefined }
}
const badJson = () => error(400, 'INVALID_JSON', '请求必须是合法 JSON 对象')
const validName = (name: unknown) => typeof name === 'string' && [...name.trim()].length >= 1 && [...name.trim()].length <= 100
let failBillingStatusOnce = true

export const handlers = [
  http.get(base, async ({ request }) => {
    await delay(450)
    const url = new URL(request.url)
    const paging = pagination(url)
    if (!paging) return invalid('pagination', 'page/page_size 必须为合法整数')
    const environment = url.searchParams.get('environment')
    const enabled = url.searchParams.get('enabled')
    if (environment && !environments.includes(environment as Flag['environment'])) return error(400, 'INVALID_ENVIRONMENT', '环境无效')
    if (enabled && enabled !== 'true' && enabled !== 'false') return invalid('enabled', '必须为 true 或 false')
    const q = (url.searchParams.get('q') ?? '').toLowerCase()
    const matches = flags.filter(flag => (!environment || flag.environment === environment)
      && (!enabled || String(flag.enabled) === enabled)
      && (flag.name.toLowerCase().includes(q) || flag.key.toLowerCase().includes(q)))
    return HttpResponse.json(paged(matches.map(flag => ({ ...flag, rule_count: rules.filter(rule => rule.flag_id === flag.id).length })), paging))
  }),
  http.post(base, async ({ request }) => {
    await delay(600)
    const input = await body(request)
    if (!input) return badJson()
    if (!validName(input.name)) return invalid('name', 'trim 后长度须为 1–100')
    if (typeof input.key !== 'string' || !/^[a-z][a-z0-9._-]{1,63}$/.test(input.key)) return error(400, 'INVALID_KEY', 'key 格式无效')
    if (!environments.includes(input.environment as Flag['environment'])) return error(400, 'INVALID_ENVIRONMENT', '环境无效')
    if (typeof input.default_value !== 'boolean') return invalid('default_value', '必须为布尔值')
    if (input.enabled !== undefined && typeof input.enabled !== 'boolean') return invalid('enabled', '必须为布尔值')
    if (input.description !== undefined && typeof input.description !== 'string') return invalid('description', '必须为字符串')
    if (flags.some(flag => flag.key === input.key && flag.environment === input.environment)) return error(409, 'DUPLICATE_FLAG_KEY', '该 key 在此环境已存在')
    const timestamp = new Date().toISOString()
    const flag: Flag = { id: crypto.randomUUID(), name: (input.name as string).trim(), key: input.key,
      environment: input.environment as Flag['environment'], default_value: input.default_value,
      description: (input.description as string) ?? '', enabled: (input.enabled as boolean) ?? true,
      created_at: timestamp, updated_at: timestamp }
    flags.unshift(flag)
    recordHistory(flag, 'flag_created', { flag: { name: flag.name, key: flag.key, environment: flag.environment, default_value: flag.default_value, enabled: flag.enabled } })
    return HttpResponse.json(flag, { status: 201 })
  }),
  http.get(`${base}/:id/history`, async ({ params, request }) => {
    await delay(350)
    if (!flags.some(flag => flag.id === params.id)) return notFound()
    const paging = pagination(new URL(request.url))
    return paging ? HttpResponse.json(paged(histories.get(String(params.id)) ?? [], paging)) : invalid('pagination', '分页参数无效')
  }),
  http.get(`${base}/:id`, async ({ params }) => {
    await delay(450)
    const flag = flags.find(flag => flag.id === params.id)
    return flag ? HttpResponse.json({ ...flag, rules: rules.filter(rule => rule.flag_id === flag.id).sort((a, b) => a.priority - b.priority) }) : notFound()
  }),
  http.patch(`${base}/:id/status`, async ({ params, request }) => {
    await delay(900)
    const flag = flags.find(flag => flag.id === params.id)
    if (!flag) return notFound()
    const input = await body(request)
    if (!input) return badJson()
    if (typeof input.enabled !== 'boolean') return invalid('enabled', '必须为布尔值')
    // Deterministic mock-only failure for the rollback acceptance check; retry succeeds.
    if (flag.key === 'billing-portal' && failBillingStatusOnce) {
      failBillingStatusOnce = false
      return error(500, 'INTERNAL_ERROR', '状态更新失败，请重试')
    }
    if (flag.enabled !== input.enabled) {
      recordHistory(flag, input.enabled ? 'flag_enabled' : 'flag_disabled', { changes: { enabled: { old: flag.enabled, new: input.enabled } } })
      flag.enabled = input.enabled
      flag.updated_at = new Date().toISOString()
    }
    return HttpResponse.json(flag)
  }),
  http.patch(`${base}/:id`, async ({ params, request }) => {
    await delay(600)
    const flag = flags.find(flag => flag.id === params.id)
    if (!flag) return notFound()
    const input = await body(request)
    if (!input) return badJson()
    if ('key' in input || 'environment' in input) return error(400, 'IMMUTABLE_FIELD', 'key 与 environment 创建后不可变')
    if ('name' in input && !validName(input.name)) return invalid('name', 'trim 后长度须为 1–100')
    if ('default_value' in input && typeof input.default_value !== 'boolean') return invalid('default_value', '必须为布尔值')
    if ('description' in input && typeof input.description !== 'string') return invalid('description', '必须为字符串')
    const changes: Record<string, unknown> = {}
    for (const field of ['name', 'description', 'default_value'] as const) {
      const value = field === 'name' && typeof input[field] === 'string' ? input[field].trim() : input[field]
      if (value !== undefined && value !== flag[field]) changes[field] = { old: flag[field], new: value }
    }
    if (Object.keys(changes).length) {
      Object.assign(flag, ...Object.entries(changes).map(([field, value]) => ({ [field]: (value as { new: unknown }).new })))
      flag.updated_at = new Date().toISOString()
      recordHistory(flag, 'flag_updated', { changes })
    }
    return HttpResponse.json(flag)
  }),
]
