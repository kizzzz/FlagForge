import { delay, http, HttpResponse } from 'msw'
import { environments } from '../api/client'
import type { BatchResult, Environment, Rule, RuleInput } from '../api/client'
import { hasNonFinite, isObject, isScalar } from '../utils/json'
import { flags, recordHistory, rules } from './data'
import { evaluate } from './evaluate'
type Helpers = {
  body: (request: Request) => Promise<{ input: Record<string, unknown> } | { response: Response }>
  error: (status: number, code: string, message: string, details?: unknown) => Response
  invalid: (field: string, reason: string) => Response
  notFound: () => Response
}
export function detailHandlers({ body, error, invalid, notFound }: Helpers) {
  const base = '/api/v1/flags'
  function validateRule(input: Record<string, unknown>) {
    if (typeof input.attribute !== 'string' || !input.attribute.trim() || [...input.attribute.trim()].length > 64) return invalid('attribute', 'trim 后须为 1–64 字符')
    if (!Number.isInteger(input.priority) || Number(input.priority) < 1) return invalid('priority', '必须是 ≥1 的整数')
    if (typeof input.roll_value !== 'boolean') return invalid('roll_value', '必须为布尔值')
    if (!['equals', 'in'].includes(String(input.operator))) return invalid('operator', '只支持 equals / in')
    const expected = input.expected_value
    if (input.operator === 'equals' ? !isScalar(expected) : !Array.isArray(expected) || !expected.length || !expected.every(isScalar)) return invalid('expected_value', 'equals 仅支持标量；in 仅支持非空标量数组')
  }
  async function saveRule(request: Request, id: string, rid?: string) {
    await delay(400)
    const flag = flags.find(f => f.id === id)
    if (!flag) return notFound()
    const existing = rid ? rules.find(r => r.id === rid && r.flag_id === id) : undefined
    if (rid && !existing) return error(404, 'RULE_NOT_FOUND', '规则不存在或不属于此 Flag')
    const parsed = await body(request)
    if ('response' in parsed) return parsed.response
    const input = { ...(existing ?? {}), ...parsed.input }
    const failure = validateRule(input)
    if (failure) return failure
    if (rules.some(r => r.flag_id === id && r.id !== rid && r.priority === input.priority)) return error(409, 'DUPLICATE_PRIORITY', '该优先级已被占用')
    const fields: RuleInput = { attribute: (input.attribute as string).trim(), operator: input.operator as Rule['operator'], expected_value: input.expected_value as Rule['expected_value'], roll_value: input.roll_value as boolean, priority: input.priority as number }
    if (existing) {
      const changes: Record<string, unknown> = {}
      for (const key of Object.keys(fields) as (keyof RuleInput)[]) if (JSON.stringify(existing[key]) !== JSON.stringify(fields[key])) changes[key] = { old: existing[key], new: fields[key] }
      if (Object.keys(changes).length) { Object.assign(existing, fields, { updated_at: new Date().toISOString() }); recordHistory(flag, 'rule_updated', { rule_id: rid, priority: existing.priority, changes }) }
      return HttpResponse.json(existing)
    }
    const timestamp = new Date().toISOString()
    const rule: Rule = { ...fields, id: crypto.randomUUID(), flag_id: id, created_at: timestamp, updated_at: timestamp }
    rules.push(rule); recordHistory(flag, 'rule_created', { rule: { ...rule } })
    return HttpResponse.json(rule, { status: 201 })
  }
  async function runEvaluation(request: Request, batch: boolean) {
    await delay(500)
    const parsed = await body(request)
    if ('response' in parsed) return parsed.response
    const { input } = parsed
    if (typeof input.flag_key !== 'string' || !input.flag_key.trim()) return invalid('flag_key', '请输入 key')
    if (!environments.includes(input.environment as Environment)) return error(400, 'INVALID_ENVIRONMENT', '环境无效')
    const list = batch ? input.user_attributes_list : [input.user_attributes]
    if (!Array.isArray(list) || !list.length) return invalid('user_attributes_list', '至少一组用户属性')
    if (list.length > 100) return error(400, 'BATCH_LIMIT_EXCEEDED', '最多 100 组')
    const index = list.findIndex(attrs => !isObject(attrs) || hasNonFinite(attrs))
    if (index >= 0) return error(400, 'INVALID_USER_ATTRIBUTES', '用户属性必须是 JSON object，数字必须有限且可序列化', batch ? { index } : {})
    const flag = flags.find(f => f.key === input.flag_key && f.environment === input.environment)
    if (!flag) return notFound()
    const evaluations = list.map(attrs => evaluate(flag, rules.filter(r => r.flag_id === flag.id), attrs))
    if (!batch) return HttpResponse.json(evaluations[0])
    const hit = new Map<string, BatchResult['summary']['hit_rules'][number]>()
    const results = evaluations.map(({ value, reason, trace }, index) => {
      if (reason.rule_id) { const rule = rules.find(r => r.id === reason.rule_id)!; const current = hit.get(rule.id); hit.set(rule.id, { rule_id: rule.id, priority: rule.priority, attribute: rule.attribute, count: (current?.count ?? 0) + 1 }) }
      return { index, value, reason, trace }
    })
    const true_count = results.filter(r => r.value).length
    return HttpResponse.json({ results, summary: { total: results.length, true_count, false_count: results.length - true_count, default_count: results.filter(r => r.reason.type === 'default').length, hit_rules: [...hit.values()].sort((a, b) => b.count - a.count || a.priority - b.priority) } })
  }
  return [
    http.post(`${base}/:id/rules`, ({ params, request }) => saveRule(request, String(params.id))),
    http.patch(`${base}/:id/rules/:rid`, ({ params, request }) => saveRule(request, String(params.id), String(params.rid))),
    http.delete(`${base}/:id/rules/:rid`, async ({ params }) => {
      await delay(350)
      const flag = flags.find(f => f.id === params.id)
      if (!flag) return notFound()
      const index = rules.findIndex(r => r.id === params.rid && r.flag_id === params.id)
      if (index < 0) return error(404, 'RULE_NOT_FOUND', '规则不存在或不属于此 Flag')
      const [rule] = rules.splice(index, 1); recordHistory(flag, 'rule_deleted', { rule })
      return new HttpResponse(null, { status: 204 })
    }),
    http.post('/api/v1/evaluate', ({ request }) => runEvaluation(request, false)),
    http.post('/api/v1/evaluate/batch', ({ request }) => runEvaluation(request, true)),
    http.post(`${base}/:id/copy`, async ({ params, request }) => {
      await delay(400)
      const source = flags.find(f => f.id === params.id)
      if (!source) return notFound()
      const parsed = await body(request)
      if ('response' in parsed) return parsed.response
      const target = parsed.input.target_environment as Environment
      if (!environments.includes(target)) return error(400, 'INVALID_ENVIRONMENT', '环境无效')
      if (target === source.environment) return error(400, 'COPY_TARGET_CONFLICT', '目标环境必须与源环境不同')
      if (flags.some(f => f.key === source.key && f.environment === target)) return error(409, 'DUPLICATE_FLAG_KEY', '目标环境已有同 key 的 Flag')
      const timestamp = new Date().toISOString()
      const flag = { ...source, id: crypto.randomUUID(), environment: target, enabled: false, created_at: timestamp, updated_at: timestamp }
      flags.unshift(flag)
      rules.push(...rules.filter(r => r.flag_id === source.id).map(r => ({ ...r, id: crypto.randomUUID(), flag_id: flag.id, expected_value: structuredClone(r.expected_value), created_at: timestamp, updated_at: timestamp })))
      recordHistory(flag, 'flag_copied', { flag: { ...flag }, source: { flag_id: source.id, environment: source.environment } })
      return HttpResponse.json(flag, { status: 201 })
    }),
  ]
}
