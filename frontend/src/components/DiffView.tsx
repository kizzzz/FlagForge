import { Card, Space, Typography } from 'antd'
import type { History } from '../api/client'
import { isObject } from '../utils/json'
import ValueDisplay from './ValueDisplay'
const labels: Record<string, string> = { name: '名称', key: 'Key', environment: '环境', default_value: '默认值', enabled: '启用状态', description: '描述', attribute: '属性', operator: '操作符', expected_value: '期望值', roll_value: '返回值', priority: '优先级' }
function Value({ value }: { value: unknown }) { return typeof value === 'boolean' ? <ValueDisplay value={value} /> : <code>{JSON.stringify(value) ?? '—'}</code> }
export default function DiffView({ history }: { history: History }) {
  const { summary, action } = history
  const rule = isObject(summary.rule) ? summary.rule : undefined
  const flag = isObject(summary.flag) ? summary.flag : undefined
  const changes = isObject(summary.changes) ? summary.changes : undefined
  return <Space direction="vertical" style={{ width: '100%' }}>
    {rule && <div className="rule-summary"><strong>#{String(rule.priority)}</strong> <code>{String(rule.attribute)} {String(rule.operator)} {JSON.stringify(rule.expected_value)}</code> → <Value value={rule.roll_value} /></div>}
    {action === 'rule_updated' && <Typography.Text type="secondary">规则 #{String(summary.priority)} · <code>{String(summary.rule_id)}</code></Typography.Text>}
    {changes && Object.entries(changes).map(([field, change]) => isObject(change) && <div key={field} className="diff-row"><strong>{labels[field] ?? field}</strong><span className="diff-old"><Value value={change.old} /></span><span>→</span><span className="diff-new"><Value value={change.new} /></span></div>)}
    {flag && <Card size="small" className="history-object">{Object.entries(flag).filter(([key]) => key in labels).map(([key, value]) => <div key={key}><Typography.Text type="secondary">{labels[key]}：</Typography.Text><Value value={value} /></div>)}</Card>}
    {isObject(summary.source) && <Typography.Text type="secondary">复制来源：{String(summary.source.environment)} · {String(summary.source.flag_id)}</Typography.Text>}
  </Space>
}
