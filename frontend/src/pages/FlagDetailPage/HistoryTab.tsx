import { useEffect, useRef, useState } from 'react'
import { Alert, Button, Empty, Space, Spin, Timeline, Typography } from 'antd'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import { flagApi } from '../../api/client'
import type { History } from '../../api/client'
import DiffView from '../../components/DiffView'
dayjs.extend(relativeTime)
const actions: Record<string, { label: string; icon: string; color: string }> = {
  flag_created: { label: '创建 Flag', icon: '＋', color: 'green' }, flag_updated: { label: '更新 Flag', icon: '✎', color: 'blue' }, flag_enabled: { label: '启用 Flag', icon: '▶', color: 'green' }, flag_disabled: { label: '停用 Flag', icon: 'Ⅱ', color: 'gray' },
  rule_created: { label: '创建规则', icon: '＋', color: 'green' }, rule_updated: { label: '更新规则', icon: '✎', color: 'blue' }, rule_deleted: { label: '删除规则', icon: '−', color: 'red' }, flag_copied: { label: '复制 Flag', icon: '⇥', color: 'blue' },
}
export default function HistoryTab({ flagId }: { flagId: string }) {
  const [items, setItems] = useState<History[]>([]), [total, setTotal] = useState(0), [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true), [failed, setFailed] = useState(false), [retry, setRetry] = useState(0)
  const busy = useRef(false)
  useEffect(() => {
    const controller = new AbortController(); busy.current = true; setLoading(true); setFailed(false)
    flagApi.history(flagId, { page, page_size: 5 }, { signal: controller.signal }).then(result => {
      if (!controller.signal.aborted) { setItems(previous => page === 1 ? result.items : [...previous, ...result.items]); setTotal(result.total) }
    }).catch(() => { if (!controller.signal.aborted) setFailed(true) }).finally(() => { if (!controller.signal.aborted) { setLoading(false); busy.current = false } })
    return () => controller.abort()
  }, [flagId, page, retry])
  return <>
    {failed && <Alert type="error" message="历史加载失败" action={<Button onClick={() => setRetry(value => value + 1)}>重试</Button>} style={{ marginBottom: 16 }} />}
    <Spin spinning={loading} tip="正在加载历史…"><div style={{ minHeight: 80 }}>
      {!items.length && !loading && !failed && <Empty description="还没有操作历史" />}
      <Timeline items={items.map(history => { const action = actions[history.action]; return { color: action?.color, dot: <span aria-label={action?.label}>{action?.icon}</span>, children: <Space direction="vertical" style={{ width: '100%' }}><Space wrap><strong>{action?.label ?? history.action}</strong><Typography.Text type="secondary">{history.operator}</Typography.Text><Typography.Text type="secondary" title={dayjs(history.created_at).format('YYYY-MM-DD HH:mm:ss')}>{dayjs(history.created_at).locale('zh-cn').fromNow()}</Typography.Text></Space><DiffView history={history} /></Space> } })} />
    </div></Spin>
    {!failed && items.length < total && <Button loading={loading} disabled={loading} onClick={() => { if (!busy.current) { busy.current = true; setPage(value => value + 1) } }}>加载更多（已加载 {items.length} / {total}）</Button>}
  </>
}
