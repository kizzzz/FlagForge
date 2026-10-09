import type { Flag, History, Rule } from '../api/client'
import { environments } from '../api/client'

const timestamp = '2026-10-09T12:00:00Z'
const seed = [
  ['结账页重构灰度', 'checkout-redesign-2026', 'production', true, false],
  ['结账页开发验证', 'checkout-redesign-2026', 'development', true, true],
  ['搜索排序实验', 'new-search-rank', 'staging', true, false],
  ['账单门户', 'billing-portal', 'development', false, true],
] as const
export const flags: Flag[] = Array.from({ length: 26 }, (_, index) => {
  const item = seed[index]
  return {
    id: `00000000-0000-4000-8000-${String(index + 1).padStart(12, '0')}`,
    name: item?.[0] ?? `功能实验 ${String(index + 1).padStart(2, '0')}`,
    key: item?.[1] ?? `feature-experiment-${index + 1}`,
    environment: item?.[2] ?? environments[index % 3],
    enabled: item?.[3] ?? index % 2 === 0,
    default_value: item?.[4] ?? false,
    description: item ? '按环境管理功能发布与灰度开关。' : '',
    created_at: timestamp,
    updated_at: timestamp,
  }
})
const ruleSeeds: Pick<Rule, 'attribute' | 'operator' | 'expected_value' | 'roll_value'>[][] = [
  [{ attribute: 'country', operator: 'in', expected_value: ['US', 'CA', 'GB'], roll_value: true }, { attribute: 'plan', operator: 'equals', expected_value: 'pro', roll_value: true }, { attribute: 'user_id', operator: 'in', expected_value: [9001, 9002, 9003], roll_value: true }],
  [{ attribute: 'country', operator: 'equals', expected_value: 'CN', roll_value: false }],
  [{ attribute: 'plan', operator: 'in', expected_value: ['pro', 'team'], roll_value: true }, { attribute: 'beta_tester', operator: 'equals', expected_value: true, roll_value: true }],
  [{ attribute: 'plan', operator: 'equals', expected_value: 'pro', roll_value: true }],
]
export const rules: Rule[] = flags.slice(0, 4).flatMap((flag, index) =>
  ruleSeeds[index].map((seedRule, ruleIndex) => ({
    id: `10000000-0000-4000-8000-${String(index * 3 + ruleIndex + 1).padStart(12, '0')}`,
    flag_id: flag.id,
    ...seedRule, priority: ruleIndex + 1, created_at: timestamp, updated_at: timestamp,
  })),
)
export const histories = new Map<string, History[]>(flags.map(flag => [flag.id, [{
  id: historiesId(), action: 'flag_created', operator: 'admin',
  summary: { flag: { name: flag.name, key: flag.key, environment: flag.environment, default_value: flag.default_value, enabled: flag.enabled } }, created_at: timestamp,
}]]))
let nextId = 100
function historiesId() { return Number.parseInt(crypto.randomUUID().slice(0, 7), 16) }
export function recordHistory(flag: Flag, action: string, summary: Record<string, unknown>) {
  const list = histories.get(flag.id) ?? []
  // Audit entries own their complete nested snapshot, independent of live rules/flags and caller mutations.
  list.unshift({ id: nextId++, action, operator: 'admin', summary: structuredClone(summary), created_at: new Date().toISOString() })
  histories.set(flag.id, list)
}

// Demonstration audit trail covers all eight contractual actions; not a claim about real seed history.
const sample = flags[0], demoRule = rules[1]
for (const [action, summary] of [
  ['flag_updated', { changes: { name: { old: '结账页试验', new: sample.name }, default_value: { old: true, new: false } } }],
  ['flag_disabled', { changes: { enabled: { old: true, new: false } } }],
  ['flag_enabled', { changes: { enabled: { old: false, new: true } } }],
  ['rule_created', { rule: demoRule }],
  ['rule_updated', { rule_id: demoRule.id, priority: 2, changes: { roll_value: { old: false, new: true } } }],
  ['rule_deleted', { rule: { ...demoRule, priority: 4, attribute: 'legacy_user' } }],
  ['flag_copied', { flag: { name: sample.name, key: sample.key, environment: sample.environment, enabled: false, default_value: false }, source: { flag_id: flags[1].id, environment: 'development' } }],
] as [string, Record<string, unknown>][]) recordHistory(sample, action, summary)
