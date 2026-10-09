import { Form, Input, Select } from 'antd'
import { environments } from '../../api/client'
export default function FlagInputs() { return <div className="console-target"><Form.Item name="flag_key" label="Flag Key" rules={[{ required: true, whitespace: true, message: '请输入 Flag Key' }]}><Input placeholder="checkout-redesign-2026" /></Form.Item><Form.Item name="environment" label="环境" rules={[{ required: true }]}><Select options={environments.map(value => ({ value, label: value }))} /></Form.Item></div> }
