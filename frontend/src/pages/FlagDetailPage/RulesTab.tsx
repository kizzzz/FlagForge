import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Popconfirm, Space, Table, Typography } from 'antd'
import { ruleApi } from '../../api/client'
import type { Rule } from '../../api/client'
import ValueDisplay from '../../components/ValueDisplay'
import RuleModal from './RuleModal'
export default function RulesTab({ flagId, rules, loading, failed, refresh }: { flagId: string; rules: Rule[]; loading: boolean; failed: boolean; refresh: () => void }) {
  const [editing, setEditing] = useState<Rule | null | undefined>()
  const [deleting, setDeleting] = useState<string>()
  const { message } = App.useApp()
  const busy = useRef(false), alive = useRef(true)
  useEffect(() => { alive.current = true; return () => { alive.current = false } }, [])
  async function remove(rule: Rule) {
    if (busy.current) return
    busy.current = true; setDeleting(rule.id)
    try { await ruleApi.remove(flagId, rule.id); if (alive.current) { void message.success('规则已删除'); refresh() } }
    catch { /* Shared reporter displays the API error. */ }
    finally { if (alive.current) { busy.current = false; setDeleting(undefined) } }
  }
  return <>
    <div className="page-heading"><Typography.Text type="secondary">数字越小越优先（1 最高），首条命中即停止；重复优先级会被拒绝。</Typography.Text><Button type="primary" disabled={loading || failed || Boolean(deleting)} onClick={() => setEditing(null)}>新建规则</Button></div>
    {failed && <Alert type="error" message="规则刷新失败，请重试" action={<Button onClick={refresh}>重试</Button>} />}
    <Table rowKey="id" loading={loading} dataSource={[...rules].sort((a, b) => a.priority - b.priority)} pagination={false} scroll={{ x: 700 }} locale={{ emptyText: '尚无规则，点击「新建规则」添加灰度条件' }} columns={[
      { title: '优先级', dataIndex: 'priority', render: value => <strong>#{value}</strong> },
      { title: '属性', dataIndex: 'attribute', render: value => <code>{value}</code> },
      { title: '操作符', dataIndex: 'operator' },
      { title: '期望值', dataIndex: 'expected_value', render: value => <code>{JSON.stringify(value)}</code> },
      { title: '结果', dataIndex: 'roll_value', render: value => <>→ <ValueDisplay value={value} /></> },
      { title: '操作', render: (_, rule) => <Space><Button size="small" disabled={loading || Boolean(deleting)} onClick={() => setEditing(rule)}>编辑</Button><Popconfirm title={`删除规则 #${rule.priority}？`} description="删除后仍可在历史中追溯。" onConfirm={() => remove(rule)} disabled={loading || Boolean(deleting)}><Button danger size="small" loading={deleting === rule.id} disabled={loading || Boolean(deleting)}>删除</Button></Popconfirm></Space> },
    ]} />
    {editing !== undefined && <RuleModal flagId={flagId} rule={editing} onClose={() => setEditing(undefined)} onSaved={refresh} />}
  </>
}
