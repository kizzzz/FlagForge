import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Card, Empty, Form, Space, Typography } from 'antd'
import dayjs from 'dayjs'
import { ApiError, evaluationApi } from '../../api/client'
import type { Evaluation, EvaluationInput } from '../../api/client'
import { evaluationCurl, parseObject, validationError } from '../../utils/json'
import JsonEditor from '../../components/JsonEditor'
import TraceTable from '../../components/TraceTable'
import ReasonText from './ReasonText'
import FlagInputs from './FlagInputs'
export default function SingleEvaluation() {
  const [form] = Form.useForm<EvaluationInput>()
  const { message } = App.useApp()
  const [text, setText] = useState('{\n  "country": "CN",\n  "plan": "pro"\n}'), [serverError, setServerError] = useState(''), [failure, setFailure] = useState('')
  const [result, setResult] = useState<Evaluation>(), [submitted, setSubmitted] = useState<EvaluationInput>(), [loading, setLoading] = useState(false)
  const controller = useRef<AbortController>(), busy = useRef(false)
  useEffect(() => () => controller.current?.abort(), [])
  const error = validationError(() => parseObject(text)) || serverError
  async function submit(values: EvaluationInput) {
    if (busy.current || validationError(() => parseObject(text))) return
    busy.current = true; setLoading(true); setFailure(''); setServerError(''); setResult(undefined)
    const request = new AbortController(); controller.current = request
    try {
      const input = { ...values, user_attributes: parseObject(text) }
      const data = await evaluationApi.evaluate(input, { signal: request.signal, inlineErrors: ['FLAG_NOT_FOUND', 'INVALID_JSON', 'INVALID_USER_ATTRIBUTES', 'VALIDATION_ERROR'] })
      if (!request.signal.aborted) { setResult(data); setSubmitted(input); void message.success('评估完成') }
    } catch (cause) {
      if (request.signal.aborted) return
      if (cause instanceof ApiError && cause.code === 'FLAG_NOT_FOUND') setFailure('Flag 不存在：检查 key 与环境，修改后重新评估。')
      else if (cause instanceof ApiError && cause.status === 400) {
        if (Array.isArray(cause.details)) form.setFields(cause.details.filter(item => ['flag_key', 'environment'].includes(item.field)).map(item => ({ name: item.field, errors: [item.reason] })))
        setServerError(cause.message)
      } else setFailure('评估失败，请检查连接后重试。')
    } finally { if (!request.signal.aborted) { busy.current = false; setLoading(false) } }
  }
  async function copy() {
    if (!submitted) return
    try { await navigator.clipboard.writeText(evaluationCurl(submitted, window.location.origin)); void message.success('curl 已复制') } catch { void message.error('复制失败，请允许剪贴板访问后重试') }
  }
  return <div className="console-grid">
    <Card title="评估输入"><Form name="single-evaluation" form={form} layout="vertical" initialValues={{ flag_key: 'checkout-redesign-2026', environment: 'production' }} disabled={loading} onFinish={values => { void submit(values) }} onValuesChange={() => { setFailure(''); setServerError('') }}>
      <FlagInputs /><JsonEditor label="用户属性 JSON" value={text} onChange={value => { setText(value); setServerError('') }} error={error} disabled={loading} hint="JSON object；仅匹配顶层属性，类型严格匹配。" />
      {failure && <Alert type="error" showIcon message={failure} style={{ marginBottom: 16 }} />}
      <Button type="primary" htmlType="submit" loading={loading} disabled={loading || Boolean(validationError(() => parseObject(text)))}> {failure ? '重试评估' : '开始评估'} </Button>
    </Form></Card>
    <Card title="评估结果" loading={loading} extra={result && <Button onClick={() => { void copy() }}>复制 curl</Button>}>
      {!result ? <Empty description="输入属性并开始评估，即可查看最终值与每条规则的轨迹" /> : <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <div><Typography.Text type="secondary">最终值 · {result.flag_key} / {result.environment}</Typography.Text><div className={`evaluation-value ${result.value ? 'value-true' : 'value-false'}`}>{String(result.value)}</div><ReasonText reason={result.reason} /></div>
        <div><Typography.Title level={5}>评估轨迹</Typography.Title><TraceTable trace={result.trace} /></div>
        <Typography.Text type="secondary">评估时间 {dayjs(result.evaluated_at).format('YYYY-MM-DD HH:mm:ss')}</Typography.Text>
      </Space>}
    </Card>
  </div>
}
