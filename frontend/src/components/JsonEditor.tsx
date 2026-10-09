import { Button, Form, Input, Typography } from 'antd'
import { parseJSON } from '../utils/json'
export default function JsonEditor({ value, onChange, error, label, hint, batch = false, disabled = false }: { value: string; onChange: (text: string) => void; error: string; label: string; hint?: string; batch?: boolean; disabled?: boolean }) {
  function format() { try { onChange(batch ? value.split('\n').map(row => row.trim() ? JSON.stringify(parseJSON(row)) : '').join('\n') : JSON.stringify(parseJSON(value), null, 2)) } catch { /* Existing validation below explains the parse failure. */ } }
  return <Form.Item label={<span>{label} <Button aria-label={`${label}格式化`} size="small" onClick={format} disabled={disabled || Boolean(error)}>格式化</Button></span>} validateStatus={error ? 'error' : undefined} help={error || undefined} extra={hint}>
    <Input.TextArea aria-label={label} aria-invalid={Boolean(error)} className="json-editor" rows={batch ? 8 : 5} value={value} onChange={event => onChange(event.target.value)} disabled={disabled} placeholder={batch ? '{"country":"CN","plan":"pro"}\n{"country":"US"}' : '{ "country": "CN", "plan": "pro" }'} />
    {batch && <Typography.Text type="secondary">{value.split('\n').filter(row => row.trim()).length} / 100 组（空行忽略）</Typography.Text>}
  </Form.Item>
}
