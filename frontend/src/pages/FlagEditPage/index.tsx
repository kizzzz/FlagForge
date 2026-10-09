import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Card, Form, Input, Radio, Select, Space, Spin, Tooltip, Typography } from 'antd'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ApiError, environments, flagApi } from '../../api/client'
import type { CreateFlag } from '../../api/client'

const initialValues = { name: '', key: '', environment: 'development', default_value: false, description: '' }
const immutableHelp = 'key 是调用方锚点，key 与 environment 创建后不可变（设计决策 D6）；需要变更时请新建 Flag。'
export default function FlagEditPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { message } = App.useApp()
  const [form] = Form.useForm<CreateFlag>()
  const [loading, setLoading] = useState(Boolean(id))
  const [saving, setSaving] = useState(false)
  const [failed, setFailed] = useState(false)
  const [reload, setReload] = useState(0)
  const submitting = useRef(false)
  const routeRequest = useRef<AbortController>()
  useEffect(() => {
    const controller = new AbortController()
    routeRequest.current = controller
    submitting.current = false
    setSaving(false)
    setFailed(false)
    form.resetFields()
    form.setFieldsValue(initialValues as CreateFlag)
    setLoading(Boolean(id))
    if (id) flagApi.get(id, { signal: controller.signal }).then(flag => {
      if (controller.signal.aborted) return
      form.setFieldsValue(flag)
    }).catch(() => { if (!controller.signal.aborted) setFailed(true) })
      .finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [id, form, reload])
  async function submit(values: CreateFlag) {
    if (submitting.current || loading || failed) return
    const routeSignal = routeRequest.current?.signal
    submitting.current = true
    setSaving(true)
    try {
      const flag = id
        ? await flagApi.update(id, { name: values.name.trim(), description: values.description ?? '', default_value: values.default_value })
        : await flagApi.create({ ...values, name: values.name.trim() }, { inlineErrors: ['DUPLICATE_FLAG_KEY'] })
      if (routeSignal?.aborted) return
      void message.success(id ? 'Flag 已保存' : 'Flag 已创建')
      navigate(`/flags/${flag.id}`)
    } catch (error) {
      if (!routeSignal?.aborted && error instanceof ApiError && error.code === 'DUPLICATE_FLAG_KEY') {
        form.setFields([{ name: 'key', errors: ['该 key 在此环境已存在'] }])
      }
    } finally {
      if (!routeSignal?.aborted) { submitting.current = false; setSaving(false) }
    }
  }
  const readonlyLabel = (text: string) => <Tooltip title={immutableHelp} trigger={['hover', 'focus']}><span tabIndex={0} className="readonly-label">{text}（创建后不可变）</span></Tooltip>
  return <Card className="edit-card">
    <div className="page-heading"><Typography.Title level={2}>{id ? '编辑 Flag' : '新建 Flag'}</Typography.Title><Link to="/">返回列表</Link></div>
    {failed && <Alert type="error" showIcon message="Flag 加载失败" description="请重试或返回列表选择其他 Flag。" action={<Button onClick={() => setReload(value => value + 1)}>重试</Button>} style={{ marginBottom: 20 }} />}
    <Spin spinning={loading} tip="正在加载 Flag…">
      <Form form={form} layout="vertical" initialValues={initialValues} disabled={loading || saving || failed} onFinish={values => { void submit(values) }}
        onValuesChange={changed => { if ('environment' in changed) form.setFields([{ name: 'key', errors: [] }]) }}>
        <Form.Item name="name" label="名称" rules={[{ required: true, whitespace: true, message: '请输入名称' }, { validator: (_, value) => !value || [...String(value).trim()].length <= 100 ? Promise.resolve() : Promise.reject(new Error('名称最多 100 个字符')) }]}>
          <Input placeholder="例如：结账页重构灰度" />
        </Form.Item>
        <Form.Item name="key" label={id ? readonlyLabel('Key') : 'Key'} extra={!id && '2–64 个字符，小写字母开头，仅限小写字母、数字、点、下划线和短横线。示例：a.b-c_d'} validateTrigger="onChange"
          rules={id ? [] : [{ required: true, message: '请输入 key' }, { pattern: /^[a-z][a-z0-9._-]{1,63}$/, message: 'key 须为 2–64 个字符，以小写字母开头，例如 a.b-c_d' }]}>
          <Input disabled={Boolean(id)} placeholder="例如：checkout-redesign-2026" className="flag-key" />
        </Form.Item>
        <Form.Item name="environment" label={id ? readonlyLabel('环境') : '环境'} rules={[{ required: true, message: '请选择环境' }]}>
          <Select disabled={Boolean(id)} options={environments.map(value => ({ label: value, value }))} />
        </Form.Item>
        <Form.Item name="default_value" label="默认值" extra="没有规则命中时返回的值；停用 Flag 始终返回 false。" rules={[{ required: true, message: '请选择默认值' }]}>
          <Radio.Group options={[{ label: 'true', value: true }, { label: 'false', value: false }]} />
        </Form.Item>
        <Form.Item name="description" label="描述"><Input.TextArea rows={3} placeholder="说明此开关的用途（可选）" /></Form.Item>
        <Space><Button type="primary" htmlType="submit" loading={saving} disabled={loading || saving || failed}>{id ? '保存修改' : '创建 Flag'}</Button><Link to="/"><Button>取消</Button></Link></Space>
      </Form>
    </Spin>
  </Card>
}
