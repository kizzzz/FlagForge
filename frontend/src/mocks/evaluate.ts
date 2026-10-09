import type { Evaluation, Flag, Rule, TraceStep } from '../api/client'
export function evaluate(flag: Flag, rules: Rule[], attrs: Record<string, unknown>): Evaluation {
  const result: Evaluation = { flag_key: flag.key, environment: flag.environment, enabled: flag.enabled, value: flag.default_value, reason: { type: 'default' }, trace: [], evaluated_at: new Date().toISOString() }
  if (!flag.enabled) return { ...result, value: false, reason: { type: 'disabled' }, trace: [{ matched: false, cause: 'short_circuited_by_disabled_flag' }] }
  for (const rule of [...rules].sort((a, b) => a.priority - b.priority)) {
    const actual = attrs[rule.attribute]
    const matched = rule.operator === 'equals' ? actual === rule.expected_value : (rule.expected_value as unknown[]).some(value => value === actual)
    const cause: TraceStep['cause'] = !Object.prototype.hasOwnProperty.call(attrs, rule.attribute) ? 'attribute_missing' : matched ? 'matched' : rule.operator === 'in' ? 'value_not_in_list' : typeof actual !== typeof rule.expected_value ? 'type_mismatch' : 'value_not_equal'
    result.trace.push({ rule_id: rule.id, priority: rule.priority, attribute: rule.attribute, operator: rule.operator, matched: cause === 'matched', cause })
    if (cause === 'matched') return { ...result, value: rule.roll_value, reason: { type: 'rule_hit', rule_id: rule.id, priority: rule.priority, summary: `${rule.attribute} ${rule.operator} ${JSON.stringify(rule.expected_value)} → ${rule.roll_value}` } }
  }
  return result
}
