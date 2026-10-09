import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Card, Col, Empty, Form, Progress, Row, Space, Statistic, Table, Typography } from 'antd'
import { ApiError, evaluationApi } from '../../api/client'
import type { BatchResult, EvaluationInput } from '../../api/client'
import { isObject, parseBatch, validationError } from '../../utils/json'
import JsonEditor from '../../components/JsonEditor'
import TraceTable from '../../components/TraceTable'
import ValueDisplay from '../../components/ValueDisplay'
import FlagInputs from './FlagInputs'
import ReasonText from './ReasonText'
const sample = ['{"country":"CN","plan":"pro"}', '{"country":"US","plan":"free"}', '{"country":"CN","plan":"pro"}', '{"country":"CN","plan":"free"}', '{}'].join('\n')
export default function BatchEvaluation() {
  const [form] = Form.useForm<EvaluationInput>(), { message } = App.useApp()
  const [text, setText] = useState(sample), [serverError, setServerError] = useState(''), [failure, setFailure] = useState('')
  const [result, setResult] = useState<BatchResult>(), [submitted, setSubmitted] = useState<ReturnType<typeof parseBatch>>(), [loading, setLoading] = useState(false)
  const busy = useRef(false), controller = useRef<AbortController>()
  useEffect(() => () => controller.current?.abort(), [])
  const localError = validationError(() => parseBatch(text)), error = localError || serverError
  async function submit(values: EvaluationInput) {
    if (busy.current || localError) return
    busy.current = true; setLoading(true); setFailure(''); setServerError(''); setResult(undefined)
    const request = new AbortController(); controller.current = request
    const parsed = parseBatch(text)
    try {
      const data = await evaluationApi.batch({ flag_key: values.flag_key, environment: values.environment, user_attributes_list: parsed.values }, { signal: request.signal, inlineErrors: ['FLAG_NOT_FOUND', 'INVALID_JSON', 'INVALID_USER_ATTRIBUTES', 'VALIDATION_ERROR', 'BATCH_LIMIT_EXCEEDED'] })
      if (!request.signal.aborted) { setResult(data); setSubmitted(parsed); void message.success('批量模拟完成') }
    } catch (cause) {
      if (request.signal.aborted) return
      if (cause instanceof ApiError && cause.code === 'FLAG_NOT_FOUND') setFailure('Flag 不存在：检查 key 与环境，修改后重新模拟。')
      else if (cause instanceof ApiError && cause.status === 400) {
        // Backend index excludes blank lines and starts at 0; show the original textarea line starting at 1.
        const index = isObject(cause.details) && typeof cause.details.index === 'number' ? cause.details.index : undefined
        setServerError(index === undefined ? cause.message : `第 ${parsed.lines[index] ?? index + 1} 行：${cause.message}`)
      } else setFailure('批量模拟失败，请检查连接后重试。')
    } finally { if (!request.signal.aborted) { busy.current = false; setLoading(false) } }
  }
  const summary = result?.summary
  return <Space direction="vertical" size="large" style={{ width: '100%' }}>
    <Card title="上线前模拟：不同用户群分别拿到什么？每行一个 JSON"><Form name="batch-evaluation" form={form} layout="vertical" initialValues={{ flag_key: 'checkout-redesign-2026', environment: 'production' }} disabled={loading} onFinish={values => { void submit(values) }} onValuesChange={() => { setFailure(''); setServerError('') }}>
      <FlagInputs /><JsonEditor batch label="批量用户属性 JSON（每行一组）" value={text} onChange={value => { setText(value); setServerError('') }} error={error} disabled={loading} hint="模拟金 V 用户 / CN 用户 / 新用户；空行忽略，整批验证通过才发送，最多 100 组。" />
      {failure && <Alert type="error" message={failure} style={{ marginBottom: 16 }} />}
      <Button type="primary" htmlType="submit" loading={loading} disabled={loading || Boolean(localError)}>{failure ? '重试模拟' : '开始模拟'}</Button>
    </Form></Card>
    <Card loading={loading} title="模拟结果">{!result || !summary ? <Empty description="开始模拟后查看统计、命中分布与每组明细" /> : <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Row gutter={16}><Col xs={24} sm={8}><Statistic title="true 数" value={summary.true_count} valueStyle={{ color: '#16776b' }} /></Col><Col xs={24} sm={8}><Statistic title="false 数" value={summary.false_count} /></Col><Col xs={24} sm={8}><Statistic title={`默认值比例（${summary.default_count}/${summary.total}）`} value={summary.total ? summary.default_count / summary.total * 100 : 0} precision={1} suffix="%" /></Col></Row>
      <div><Typography.Title level={5}>命中规则分布</Typography.Title>{!summary.hit_rules.length ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="没有规则命中" /> : [...summary.hit_rules].sort((a, b) => b.count - a.count || a.priority - b.priority).map(hit => <div key={hit.rule_id} className="distribution-row"><Typography.Text>#{hit.priority} {hit.attribute} · {result.results.find(row => row.reason.rule_id === hit.rule_id)?.reason.summary ?? ''} · {hit.count} 次</Typography.Text><Progress percent={hit.count / summary.total * 100} format={percent => `${percent?.toFixed(1)}%`} /></div>)}</div>
      <Table rowKey="index" size="small" dataSource={result.results} pagination={{ pageSize: 10 }} scroll={{ x: 700 }} expandable={{ expandedRowRender: row => <TraceTable trace={row.trace} /> }} columns={[
        { title: '原始行号', dataIndex: 'index', render: index => submitted?.lines[index] ?? index + 1 },
        { title: '用户属性', dataIndex: 'index', render: index => <code>{JSON.stringify(submitted?.values[index])}</code> },
        { title: '最终值', dataIndex: 'value', render: value => <ValueDisplay value={value} /> },
        { title: '原因', dataIndex: 'reason', render: reason => <ReasonText reason={reason} /> },
      ]} />
    </Space>}</Card>
  </Space>
}
