import { useEffect, useState } from 'react'
import { Alert, Button, Card, Descriptions, Space, Spin, Tabs, Tag, Typography } from 'antd'
import { Link, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import { flagApi } from '../../api/client'
import type { FlagDetail } from '../../api/client'
import ValueDisplay from '../../components/ValueDisplay'
import RulesTab from './RulesTab'
import HistoryTab from './HistoryTab'
import CopyFlag from './CopyFlag'
export default function FlagDetailPage() {
  const { id = '' } = useParams()
  const [flag, setFlag] = useState<FlagDetail>(), [loading, setLoading] = useState(true), [failed, setFailed] = useState(false)
  const [revision, setRevision] = useState(0), [tab, setTab] = useState('rules')
  useEffect(() => { setTab('rules'); setFlag(undefined) }, [id])
  useEffect(() => {
    const controller = new AbortController(); setLoading(true); setFailed(false)
    flagApi.get(id, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) setFlag(result) })
      .catch(() => { if (!controller.signal.aborted) setFailed(true) }).finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [id, revision])
  const refresh = () => setRevision(value => value + 1)
  return <>
    <div className="page-heading"><Typography.Title level={2}>Flag 详情</Typography.Title><Space><Link to="/">返回列表</Link>{flag && <Link to={`/flags/${id}/edit`}><Button>编辑基本信息</Button></Link>}</Space></div>
    {failed && <Alert type="error" message="Flag 加载失败或不存在" description="可重试，或返回列表选择其他 Flag。" action={<Button onClick={refresh}>重试</Button>} style={{ marginBottom: 20 }} />}
    <Spin spinning={loading && !flag} tip="正在加载 Flag…"><Card style={{ minHeight: 100, marginBottom: 20 }}>
      {flag && <Descriptions title={flag.name} column={{ xs: 1, sm: 2, lg: 3 }} items={[
        { key: 'id', label: 'ID', children: <code>{flag.id}</code> }, { key: 'key', label: 'Key', children: <Typography.Text code copyable>{flag.key}</Typography.Text> }, { key: 'env', label: '环境', children: <Tag>{flag.environment}</Tag> },
        { key: 'enabled', label: '状态', children: <Tag color={flag.enabled ? 'green' : 'default'}>{flag.enabled ? '已启用' : '已停用'}</Tag> }, { key: 'default', label: '默认值', children: <ValueDisplay value={flag.default_value} /> }, { key: 'name', label: '名称', children: flag.name },
        { key: 'created', label: '创建时间', children: dayjs(flag.created_at).format('YYYY-MM-DD HH:mm:ss') }, { key: 'updated', label: '更新时间', children: dayjs(flag.updated_at).format('YYYY-MM-DD HH:mm:ss') }, { key: 'description', label: '描述', children: flag.description || '暂无描述', span: 3 },
      ]} />}
      {flag && <div style={{ marginTop: 16 }}><CopyFlag key={id} flag={flag} /></div>}
    </Card></Spin>
    {flag && <Card><Tabs activeKey={tab} onChange={setTab} items={[
      { key: 'rules', label: `规则（${flag.rules.length}）`, children: <RulesTab key={id} flagId={id} rules={flag.rules} loading={loading} failed={failed} refresh={refresh} /> },
      { key: 'history', label: '操作历史', children: <HistoryTab key={`${id}-${revision}`} flagId={id} /> },
    ]} /></Card>}
  </>
}
