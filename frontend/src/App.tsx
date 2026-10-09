import { useLayoutEffect } from 'react'
import { App as AntApp, ConfigProvider, Layout, Menu, Result, Typography } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { BrowserRouter, Link, Route, Routes, useLocation } from 'react-router-dom'
import { setApiErrorReporter } from './api/client'
import FlagListPage from './pages/FlagListPage'
import FlagEditPage from './pages/FlagEditPage'
import './App.css'

function Shell() {
  const { message } = AntApp.useApp()
  const location = useLocation()
  useLayoutEffect(() => {
    setApiErrorReporter(error => { void message.error(error.message) })
    return () => setApiErrorReporter(() => undefined)
  }, [message])
  return <Layout className="app-shell">
    <Layout.Header className="app-header">
      <Link className="brand" to="/">FlagForge</Link>
      <Menu mode="horizontal" selectedKeys={[location.pathname === '/console' ? 'console' : 'flags']} items={[
        { key: 'flags', label: <Link to="/">Flag 列表</Link> },
        { key: 'console', label: <Link to="/console">评估控制台</Link> },
      ]} />
    </Layout.Header>
    <Layout.Content className="app-content">
      <Routes>
        <Route path="/" element={<FlagListPage />} />
        <Route path="/flags/new" element={<FlagEditPage />} />
        <Route path="/flags/:id/edit" element={<FlagEditPage />} />
        <Route path="/flags/:id" element={<Result title="Flag 详情" subTitle="T4b 实现" extra={<Link to="/">返回列表</Link>} />} />
        <Route path="/console" element={<Result title="评估控制台" subTitle="T4b 实现" />} />
        <Route path="*" element={<Result status="404" title="页面不存在" extra={<Link to="/">返回列表</Link>} />} />
      </Routes>
    </Layout.Content>
    <Layout.Footer><Typography.Text type="secondary">FlagForge · 按环境管理功能发布</Typography.Text></Layout.Footer>
  </Layout>
}
export default function App() {
  return <ConfigProvider locale={zhCN} componentSize="middle" theme={{ token: { colorPrimary: '#16776b', borderRadius: 6 } }}>
    <AntApp><BrowserRouter><Shell /></BrowserRouter></AntApp>
  </ConfigProvider>
}
