import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Card, Empty, Input, Select, Space, Switch, Table, Tag, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { Link } from 'react-router-dom'
import { environments, flagApi } from '../../api/client'
import type { Environment, FlagListItem, ListFlagsQuery, Paged } from '../../api/client'

const colors = { development: 'cyan', staging: 'gold', production: 'purple' }
const emptyPage: Paged<FlagListItem> = { items: [], page: 1, page_size: 20, total: 0 }
export default function FlagListPage() {
  const { message } = App.useApp()
  const [query, setQuery] = useState<ListFlagsQuery>({ page: 1, page_size: 20 })
  const [search, setSearch] = useState('')
  const [data, setData] = useState(emptyPage)
  const [loading, setLoading] = useState(true)
  const [failed, setFailed] = useState(false)
  const [reload, setReload] = useState(0)
  const [pending, setPending] = useState<Record<string, boolean>>({})
  const pendingRef = useRef<Record<string, boolean>>({})
  const mounted = useRef(true)
  useEffect(() => { mounted.current = true; return () => { mounted.current = false } }, [])
  useEffect(() => {
    const timer = setTimeout(() => setQuery(previous => previous.q === search ? previous : { ...previous, q: search, page: 1 }), 300)
    return () => clearTimeout(timer)
  }, [search])
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setFailed(false)
    flagApi.list(query, { signal: controller.signal }).then(result => {
      if (controller.signal.aborted) return
      setData({ ...result, items: result.items.map(row => ({ ...row, enabled: pendingRef.current[row.id] ?? row.enabled })) })
    }).catch(() => { if (!controller.signal.aborted) setFailed(true) })
      .finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [query, reload])
  const filter = (update: Partial<ListFlagsQuery>) => setQuery(previous => ({ ...previous, ...update, page: 1 }))
  async function toggle(row: FlagListItem, enabled: boolean) {
    if (row.id in pendingRef.current) return
    const before = row.enabled
    pendingRef.current[row.id] = enabled
    setPending({ ...pendingRef.current })
    setData(previous => ({ ...previous, items: previous.items.map(item => item.id === row.id ? { ...item, enabled } : item) }))
    try {
      const updated = await flagApi.setStatus(row.id, enabled)
      if (!mounted.current) return
      setData(previous => ({ ...previous, items: previous.items.map(item => item.id === row.id ? { ...item, ...updated } : item) }))
      void message.success(enabled ? 'Flag 已启用' : 'Flag 已停用')
      setReload(value => value + 1)
    } catch {
      if (mounted.current) setData(previous => ({ ...previous, items: previous.items.map(item => item.id === row.id ? { ...item, enabled: before } : item) }))
    } finally {
      delete pendingRef.current[row.id]
      if (mounted.current) setPending({ ...pendingRef.current })
    }
  }
  const columns: ColumnsType<FlagListItem> = [
    { title: '名称', dataIndex: 'name', width: 180, render: (name, row) => <Link to={`/flags/${row.id}`}>{name}</Link> },
    { title: 'Key', dataIndex: 'key', width: 240, render: key => <Typography.Text className="flag-key" copyable={{ text: key }}>{key}</Typography.Text> },
    { title: '环境', dataIndex: 'environment', width: 135, render: (env: Environment) => <Tag color={colors[env]}>{env}</Tag> },
    { title: '启用', dataIndex: 'enabled', width: 90, render: (enabled: boolean, row) => <Switch aria-label={`${row.name}启用状态`} checked={enabled} loading={row.id in pending} onChange={value => { void toggle(row, value) }} /> },
    { title: '默认值', dataIndex: 'default_value', width: 90, render: value => <Tag>{String(value)}</Tag> },
    { title: '规则数', dataIndex: 'rule_count', width: 80 },
    { title: '更新时间', dataIndex: 'updated_at', width: 170, render: value => dayjs(value).format('YYYY-MM-DD HH:mm') },
    { title: '操作', width: 105, render: (_, row) => <Space><Link to={`/flags/${row.id}`}>详情</Link><Link to={`/flags/${row.id}/edit`}>编辑</Link></Space> },
  ]
  return <>
    <div className="page-heading"><div><Typography.Title level={2}>Flag 列表</Typography.Title><Typography.Text type="secondary">按环境查看与管理功能开关</Typography.Text></div><Link to="/flags/new"><Button type="primary">新建 Flag</Button></Link></div>
    <Card>
      <div className="filter-bar">
        <Select<Environment | 'all'> aria-label="环境筛选" value={query.environment ?? 'all'} onChange={value => filter({ environment: value === 'all' ? undefined : value })} options={[{ value: 'all', label: '全部环境' }, ...environments.map(env => ({ value: env, label: env }))]} />
        <Select aria-label="启用状态筛选" value={query.enabled === undefined ? 'all' : String(query.enabled)} onChange={value => filter({ enabled: value === 'all' ? undefined : value === 'true' })} options={[{ value: 'all', label: '全部启用状态' }, { value: 'true', label: '已启用' }, { value: 'false', label: '已停用' }]} />
        <Input aria-label="搜索名称或 key" placeholder="搜索名称或 key" value={search} allowClear onChange={event => setSearch(event.target.value)} />
      </div>
      {failed && <Alert showIcon type="error" message="列表加载失败" description="请检查网络连接后重试。" action={<Button onClick={() => setReload(value => value + 1)}>重试</Button>} style={{ marginBottom: 16 }} />}
      <Table rowKey="id" columns={columns} dataSource={failed ? [] : data.items} loading={loading} scroll={{ x: 1120 }}
        locale={{ emptyText: <Empty description={failed ? '暂时无法获取 Flag' : '没有匹配的 Flag，试试调整筛选条件或创建新 Flag'}>{!failed && <Link to="/flags/new"><Button type="primary">新建 Flag</Button></Link>}</Empty> }}
        pagination={{ current: query.page, pageSize: query.page_size, total: failed ? 0 : data.total, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showTotal: total => `共 ${total} 个 Flag`, onChange: (page, page_size) => setQuery(previous => ({ ...previous, page: previous.page_size === page_size ? page : 1, page_size })) }} />
    </Card>
  </>
}
