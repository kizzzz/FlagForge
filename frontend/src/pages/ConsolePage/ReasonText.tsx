import { Tag } from 'antd'
import type { Reason } from '../../api/client'
function reasonText(reason: Reason) {
  if (reason.type === 'disabled') return 'Flag 已停用，短路返回 false'
  if (reason.type === 'default') return '未命中规则，返回默认值'
  return `命中 #${reason.priority} · ${reason.summary ?? ''}`
}
export default function ReasonText({ reason }: { reason: Reason }) { return <Tag color={reason.type === 'rule_hit' ? 'green' : reason.type === 'default' ? 'blue' : 'default'}>{reasonText(reason)}</Tag> }
