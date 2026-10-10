import { delay, http, HttpResponse } from 'msw'
import { environments } from '../api/client'
import type { Flag } from '../api/client'
import { flags, histories, recordHistory, rules } from './data'
import { detailHandlers } from './detailHandlers'

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
function jsonErrorLocation(raw: string, reason: string) {
  const position = reason.match(/\bposition\s+(\d+)/i)
  const lineColumn = reason.match(/\bline\s+(\d+)\s+column\s+(\d+)/i)
  let offset = position ? Number(position[1]) : undefined
  if (offset === undefined && lineColumn) {
    const lines = raw.split('\n')
    const line = Number(lineColumn[1])
    if (line >= 1 && line <= lines.length) offset = lines.slice(0, line - 1).reduce((length, text) => length + text.length + 1, 0) + Number(lineColumn[2]) - 1
  }
  if (offset === undefined && /unexpected end|end of (?:json|data)/i.test(reason)) offset = raw.length
  if (offset === undefined) {
    // Some V8 errors name a token without its offset. Only infer a unique occurrence.
    const token = reason.match(/^Unexpected token '([^']+)'/i)?.[1]
    if (token && raw.indexOf(token) >= 0 && raw.indexOf(token) === raw.lastIndexOf(token)) offset = raw.indexOf(token)
  }
  if (offset === undefined || offset < 0 || offset > raw.length) {
    return { message: `解析器未提供精确错误位置；输入位置范围 0–${raw.length}`, details: { input_length: raw.length, offset_unit: 'UTF-16', position: null } }
  }
  const prefix = raw.slice(0, offset).split('\n')
  const line = prefix.length
  const column = prefix[prefix.length - 1].length + 1
  return { message: `位置 ${offset}（第 ${line} 行，第 ${column} 列；UTF-16 偏移从 0 开始）`, details: { position: offset, line, column, offset_unit: 'UTF-16' } }
}
async function body(request: Request): Promise<{ input: Record<string, unknown> } | { response: Response }> {
  const raw = await request.text()
  try {
    const value: unknown = JSON.parse(raw)
    return value && typeof value === 'object' && !Array.isArray(value)
      ? { input: value as Record<string, unknown> }
      : { response: invalid('body', '请求 JSON 必须是对象，不能是数组、null 或标量') }
  } catch (cause) {
    const reason = cause instanceof Error ? cause.message : 'JSON 解析失败'
    const location = jsonErrorLocation(raw, reason)
    return { response: error(400, 'INVALID_JSON', `JSON 语法错误：${location.message}。${reason}`, location.details) }
  }
}
const validName = (name: unknown) => typeof name === 'string' && [...name.trim()].length >= 1 && [...name.trim()].length <= 100
let failBillingStatusOnce = true

export const handlers = [
  ...detailHandlers({ body, error, invalid, notFound }),
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
    const parsed = await body(request)
    if ('response' in parsed) return parsed.response
    const { input } = parsed
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
  http.delete(`${base}/:id`, async ({ params }) => {
    await delay(450)
    const id = String(params.id)
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id)) return error(400, 'VALIDATION_ERROR', 'request validation failed', [{ field: 'id', reason: 'must be a UUID in xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format' }])
    const index = flags.findIndex(flag => flag.id === id)
    if (index < 0) return notFound()
    flags.splice(index, 1)
    for (let index = rules.length - 1; index >= 0; index -= 1) { if (rules[index].flag_id === id) rules.splice(index, 1) }
    histories.delete(id)
    return new HttpResponse(null, { status: 204 })
  }),
  http.patch(`${base}/:id/status`, async ({ params, request }) => {
    await delay(900)
    const flag = flags.find(flag => flag.id === params.id)
    if (!flag) return notFound()
    const parsed = await body(request)
    if ('response' in parsed) return parsed.response
    const { input } = parsed
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
    const parsed = await body(request)
    if ('response' in parsed) return parsed.response
    const { input } = parsed
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
