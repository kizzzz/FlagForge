import { useEffect, useRef, useState } from 'react'
import { App, Button, Select, Space } from 'antd'
import { useNavigate } from 'react-router-dom'
import { ApiError, copyFlag, environments } from '../../api/client'
import type { Flag, Environment } from '../../api/client'
export default function CopyFlag({ flag }: { flag: Flag }) {
  const [target, setTarget] = useState<Environment>(), [saving, setSaving] = useState(false)
  const { modal, message } = App.useApp(), navigate = useNavigate()
  const busy = useRef(false), alive = useRef(true)
  useEffect(() => { alive.current = true; return () => { alive.current = false } }, [])
  function confirm() {
    if (!target || busy.current) return
    const destination = target
    modal.confirm({ title: '复制环境配置', content: `将 ${flag.key} 的完整规则复制到 ${destination}。复制后新 Flag 为停用状态（enabled=false）。`, okText: '确认复制', onOk: async () => {
      if (busy.current) return
      busy.current = true; setSaving(true)
      try {
        const copied = await copyFlag(flag.id, destination, { inlineErrors: ['DUPLICATE_FLAG_KEY'] })
        if (alive.current) { void message.success('配置已复制，新 Flag 为停用状态'); navigate(`/flags/${copied.id}`) }
      } catch (cause) {
        if (alive.current && cause instanceof ApiError && cause.code === 'DUPLICATE_FLAG_KEY') void message.error('目标环境已有同 key 的 Flag，请选择其他环境')
        throw cause
      } finally { if (alive.current) { busy.current = false; setSaving(false) } }
    } })
  }
  return <Space wrap><Select aria-label="复制目标环境" placeholder="复制到其他环境" value={target} style={{ width: 180 }} disabled={saving} onChange={setTarget} options={environments.filter(env => env !== flag.environment).map(value => ({ value, label: value }))} /><Button loading={saving} disabled={!target || saving} onClick={confirm}>复制配置</Button></Space>
}
