import { useEffect, useRef, useState } from 'react'
import { App, Form, Input, InputNumber, Modal, Radio } from 'antd'
import { ApiError, ruleApi } from '../../api/client'
import type { Rule, RuleInput } from '../../api/client'
import { parseExpected, validationError } from '../../utils/json'
import JsonEditor from '../../components/JsonEditor'
export default function RuleModal({ flagId, rule, onClose, onSaved }: { flagId: string; rule: Rule | null; onClose: () => void; onSaved: () => void }) {
  const [form] = Form.useForm<RuleInput>()
  const { message } = App.useApp()
  const [expected, setExpected] = useState(rule ? JSON.stringify(rule.expected_value, null, 2) : '"pro"')
  const [saving, setSaving] = useState(false)
  const busy = useRef(false), alive = useRef(true)
  useEffect(() => { alive.current = true; return () => { alive.current = false } }, [])
  const operator = Form.useWatch('operator', form) ?? rule?.operator ?? 'equals'
  const error = validationError(() => parseExpected(expected, operator))
  async function save() {
    if (busy.current || error) return
    let values: RuleInput
    try { values = await form.validateFields() } catch { return }
    if (busy.current) return
    busy.current = true; setSaving(true)
    try {
      const input = { ...values, attribute: values.attribute.trim(), expected_value: parseExpected(expected, values.operator) }
      const options = { inlineErrors: ['DUPLICATE_PRIORITY', 'VALIDATION_ERROR'] }
      if (rule) await ruleApi.update(flagId, rule.id, input, options); else await ruleApi.create(flagId, input, options)
      if (!alive.current) return
      void message.success(rule ? '规则已保存' : '规则已创建'); onSaved(); onClose()
    } catch (cause) {
      if (!alive.current) return
      if (cause instanceof ApiError && cause.code === 'DUPLICATE_PRIORITY') form.setFields([{ name: 'priority', errors: ['该优先级已被占用'] }])
      else if (cause instanceof ApiError && cause.code === 'VALIDATION_ERROR') {
        if (Array.isArray(cause.details)) form.setFields(cause.details.map(item => ({ name: item.field, errors: [item.reason] })))
        void message.error(cause.message)
      }
    } finally { if (alive.current) { busy.current = false; setSaving(false) } }
  }
  return <Modal open title={rule ? '编辑规则' : '新建规则'} onCancel={onClose} onOk={() => { void save() }} confirmLoading={saving} okButtonProps={{ disabled: saving || Boolean(error) }} cancelButtonProps={{ disabled: saving }} closable={!saving} maskClosable={!saving}>
    <Form form={form} layout="vertical" initialValues={rule ?? { attribute: '', operator: 'equals', priority: 1, roll_value: true }} disabled={saving} onValuesChange={changed => { if ('priority' in changed) form.setFields([{ name: 'priority', errors: [] }]) }}>
      <Form.Item name="attribute" label="属性" rules={[{ required: true, whitespace: true, message: '请输入属性' }, { validator: (_, value) => [...String(value ?? '').trim()].length <= 64 ? Promise.resolve() : Promise.reject(new Error('属性最多 64 字符')) }]}><Input placeholder="例如 plan（仅顶层属性）" /></Form.Item>
      <Form.Item name="operator" label="操作符"><Radio.Group options={['equals', 'in']} /></Form.Item>
      <JsonEditor label="期望值 JSON" value={expected} onChange={setExpected} error={error} disabled={saving} hint={operator === 'equals' ? 'JSON 标量，如 "pro" / 42 / true；类型严格匹配' : '非空标量数组，如 ["US","CA"]；禁止 null 和嵌套值'} />
      <Form.Item name="priority" label="优先级" extra="数字越小越优先（1 最高）；重复优先级会被拒绝。" rules={[{ required: true, message: '请输入优先级' }, { type: 'integer', min: 1, message: '必须是 ≥1 的整数' }]}><InputNumber min={1} /></Form.Item>
      <Form.Item name="roll_value" label="命中时返回"><Radio.Group options={[{ label: 'true', value: true }, { label: 'false', value: false }]} /></Form.Item>
    </Form>
  </Modal>
}
