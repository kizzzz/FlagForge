import { Tag } from 'antd'
export default function ValueDisplay({ value }: { value: boolean }) { return <Tag color={value ? 'green' : 'default'}>{String(value)}</Tag> }
