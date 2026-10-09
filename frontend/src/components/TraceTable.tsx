import { Table, Tag } from 'antd'
import type { TraceStep } from '../api/client'
import { causeText } from '../constants/causeText'
export default function TraceTable({ trace }: { trace: TraceStep[] }) {
  return <Table size="small" pagination={false} dataSource={trace} rowKey={row => row.rule_id ?? 'disabled'} rowClassName={row => row.matched ? 'trace-hit' : ''} locale={{ emptyText: '没有规则参与评估，返回默认值' }} scroll={{ x: 550 }} columns={[
    { title: '优先级', dataIndex: 'priority', render: value => value ? `#${value}` : '—' },
    { title: '属性', dataIndex: 'attribute', render: value => <code>{value ?? '—'}</code> },
    { title: '操作符', dataIndex: 'operator', render: value => value ?? '—' },
    { title: '是否命中', dataIndex: 'matched', render: value => <Tag color={value ? 'green' : 'default'}>{value ? '命中' : '跳过'}</Tag> },
    { title: '原因', dataIndex: 'cause', render: (value: TraceStep['cause']) => causeText[value] },
  ]} />
}
