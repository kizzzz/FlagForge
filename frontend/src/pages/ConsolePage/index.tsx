import { Tabs, Typography } from 'antd'
import SingleEvaluation from './SingleEvaluation'
import BatchEvaluation from './BatchEvaluation'
export default function ConsolePage() { return <><Typography.Title level={2}>评估控制台</Typography.Title><Typography.Paragraph type="secondary">查看最终值，也看清每条规则为什么命中或跳过。数字越小越优先，首条命中即停止；停用始终返回 false。</Typography.Paragraph><Tabs items={[{ key: 'single', label: '单次评估', children: <SingleEvaluation /> }, { key: 'batch', label: '批量模拟', children: <BatchEvaluation /> }]} /></> }
