import { useCallback, useEffect, useRef, useState } from 'react'
import { AutoComplete, Button, Form, Select, Space, Tag, Typography } from 'antd'
import { environments, flagApi } from '../../api/client'
import type { Environment, FlagListItem } from '../../api/client'

export default function FlagInputs() {
  const form = Form.useFormInstance()
  const environment = Form.useWatch<Environment>('environment', form)
  const [flags, setFlags] = useState<FlagListItem[]>([])
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false), [failed, setFailed] = useState(false)
  const controller = useRef<AbortController>()
  const refresh = useCallback(async () => {
    controller.current?.abort()
    if (!environment) return
    const request = new AbortController(); controller.current = request
    setLoading(true); setFailed(false); setFlags([])
    try {
      const items: FlagListItem[] = []
      let page = 1, total = 0
      do {
        const result = await flagApi.list({ environment, page, page_size: 100 }, { signal: request.signal, inlineErrors: ['NETWORK_ERROR', 'INTERNAL_ERROR'] })
        if (request.signal.aborted) return
        items.push(...result.items); total = result.total; page += 1
        if (!result.items.length) break
      } while (items.length < total)
      if (!request.signal.aborted) setFlags(items)
    } catch {
      if (!request.signal.aborted) setFailed(true)
    } finally {
      if (!request.signal.aborted) setLoading(false)
    }
  }, [environment])
  useEffect(() => { void refresh(); return () => controller.current?.abort() }, [refresh])
  const matches = (flag: FlagListItem) => `${flag.key} ${flag.name}`.toLowerCase().includes(search.toLowerCase())
  const feedback = loading ? '正在加载 Flag 候选…' : failed ? <Space>候选加载失败，仍可自定义输入<Button size="small" onMouseDown={event => event.preventDefault()} onClick={() => { void refresh() }}>重试</Button></Space> : !flags.length ? '此环境暂无 Flag，可自定义输入 Key' : !flags.some(matches) ? '没有匹配的 Flag，可继续使用自定义 Key' : undefined
  return <div className="console-target">
    <Form.Item name="flag_key" label="Flag Key" extra={feedback} rules={[{ required: true, whitespace: true, message: '请输入 Flag Key' }]}>
      <AutoComplete placeholder="输入或选择 Flag Key" open={open} onOpenChange={setOpen} onBlur={() => setOpen(false)} onFocus={() => { setOpen(true); setSearch(''); void refresh() }} onSearch={setSearch}
        options={flags.map(flag => ({ value: flag.key, search: `${flag.key} ${flag.name}`.toLowerCase(), label: <Space><Typography.Text>{flag.key}</Typography.Text><Typography.Text type="secondary">{flag.name}</Typography.Text><Tag color={flag.enabled ? 'green' : 'default'}>{flag.enabled ? '已启用' : '已停用'}</Tag></Space> }))}
        filterOption={(_, option) => Boolean(option?.search.includes(search.toLowerCase()))} />
    </Form.Item>
    <Form.Item name="environment" label="环境" rules={[{ required: true }]}><Select options={environments.map(value => ({ value, label: value }))} /></Form.Item>
  </div>
}
