import assert from 'node:assert/strict'
import { after, afterEach, test } from 'node:test'
import React, { act } from 'react'
import { createRoot } from 'react-dom/client'
import { App as AntApp, ConfigProvider, Form } from 'antd'
import { MemoryRouter } from 'react-router-dom'
import App from '../src/App'
import FlagInputs from '../src/pages/ConsolePage/FlagInputs'
import FlagListPage from '../src/pages/FlagListPage'
import { setupServer } from 'msw/node'
import { handlers } from '../src/mocks/handlers'
import { flags, histories, rules } from '../src/mocks/data'

after(() => window.close())
let root, host
const nativeFetch = globalThis.fetch
const requests = []
const flag = (key, environment = 'production', enabled = true) => ({ id: `id-${key}`, key, name: `名称 ${key}`, environment, enabled, default_value: false, rule_count: 0, created_at: '2026-10-10T00:00:00Z', updated_at: '2026-10-10T00:00:00Z' })
const page = (items = [], total = items.length, current = 1, size = 100) => ({ items, total, page: current, page_size: size })
const response = (body, status = 200) => new Response(status === 204 ? null : JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
function network(handler) {
  requests.length = 0
  globalThis.fetch = async (path, init) => {
    const request = { url: new URL(path, window.location.origin), method: init.method ?? 'GET', body: init.body && JSON.parse(init.body), signal: init.signal }
    requests.push(request)
    return handler(request)
  }
}
async function mount(node) {
  host = document.createElement('div'); document.body.append(host); root = createRoot(host)
  await act(async () => { root.render(<ConfigProvider theme={{ token: { motion: false } }}><AntApp>{node}</AntApp></ConfigProvider>) })
}
async function settle(ms = 25) { await act(async () => { await new Promise(resolve => setTimeout(resolve, ms)) }) }
async function waitFor(predicate, description) {
  for (let i = 0; i < 60; i++) { if (predicate()) return; await settle() }
  assert.ok(predicate(), description)
}
async function click(element) { assert.ok(element, 'click target exists'); await act(async () => element.dispatchEvent(new MouseEvent('click', { bubbles: true }))) }
async function input(element, value) {
  assert.ok(element, 'input exists')
  await act(async () => { Object.getOwnPropertyDescriptor(element instanceof window.HTMLTextAreaElement ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype, 'value').set.call(element, value); element.dispatchEvent(new Event('input', { bubbles: true })) })
}
const button = (text, scope = document) => [...scope.querySelectorAll('button')].find(item => item.textContent.replace(/\s/g, '').includes(text))
const keyInput = () => host.querySelector('input[id$="flag_key"]')
async function focusKey() { await act(async () => { keyInput().focus() }); await settle() }
async function chooseOption(select, text) {
  const selection = select.querySelector('.ant-select-selector')
  await act(async () => selection.dispatchEvent(new MouseEvent('mousedown', { bubbles: true })))
  await settle()
  await click([...document.querySelectorAll('.ant-select-item-option')].find(item => item.textContent === text))
  await settle()
}
async function chooseEnvironment(env) { await chooseOption(host.querySelector('[id$="environment"]').closest('.ant-select'), env) }
afterEach(async () => {
  if (root) await act(async () => root.unmount())
  root = undefined; host?.remove(); document.body.innerHTML = ''; window.history.replaceState({}, '', '/console')
  await settle()
})
const inputs = () => <Form initialValues={{ flag_key: 'custom-key', environment: 'production' }}><FlagInputs /></Form>

test('console retains actual drafts, results, errors, batch tab/page and expanded trace through navigation; new mount resets', async () => {
  network(({ url, body }) => {
    if (url.pathname === '/api/v1/evaluate') return response({ flag_key: body.flag_key, environment: body.environment, value: true, reason: { type: 'default' }, trace: [], evaluated_at: '2026-10-10T00:00:00Z' })
    if (url.pathname.endsWith('/evaluate/batch')) return response({ summary: { total: 12, true_count: 12, false_count: 0, default_count: 12, hit_rules: [] }, results: body.user_attributes_list.map((_, index) => ({ index, value: true, reason: { type: 'default' }, trace: [] })) })
    return response(page([], 0, Number(url.searchParams.get('page') ?? 1), Number(url.searchParams.get('page_size') ?? 20)))
  })
  await mount(<App />)
  await input(keyInput(), 'my-custom-key')
  await input(host.querySelector('textarea'), '{"saved":true}')
  await click(button('开始评估'))
  await waitFor(() => host.textContent.includes('最终值 · my-custom-key'), 'single result rendered')
  await input(host.querySelector('textarea'), '{bad')
  await click([...host.querySelectorAll('[role="tab"]')].find(item => item.textContent === '批量模拟'))
  await input(host.querySelector('#batch-evaluation_flag_key'), 'batch-custom')
  await chooseOption(host.querySelector('#batch-evaluation_environment').closest('.ant-select'), 'staging')
  await input(host.querySelector('textarea[aria-label^="批量"]'), Array.from({ length: 12 }, (_, i) => JSON.stringify({ i })).join('\n'))
  await click(button('开始模拟'))
  await waitFor(() => host.querySelector('.ant-pagination-item-2'), 'batch second page available')
  assert.ok(host.textContent.includes('评估目标 · batch-custom / staging'), 'batch result identifies submitted key/environment')
  assert.ok(host.textContent.includes('模拟完成时间'), 'batch result identifies completion time')
  await input(host.querySelector('#batch-evaluation_flag_key'), 'edited-after-result')
  assert.ok(host.textContent.includes('评估目标 · batch-custom / staging'), 'old result retains original target after editing the form')
  await click(host.querySelector('.ant-pagination-item-2'))
  await click(host.querySelector('.ant-table-row-expand-icon'))
  const evaluationCount = requests.filter(item => item.method === 'POST').length
  await click([...host.querySelectorAll('a')].find(item => item.textContent === 'Flag 列表'))
  await click([...host.querySelectorAll('a')].find(item => item.textContent === '评估控制台'))
  assert.equal(host.querySelector('[role="tab"][aria-selected="true"]').textContent, '批量模拟')
  assert.equal(host.querySelector('#batch-evaluation_flag_key').value, 'edited-after-result')
  assert.ok(host.querySelector('.ant-pagination-item-2.ant-pagination-item-active'))
  assert.ok(host.querySelector('.ant-table-expanded-row'))
  assert.equal(requests.filter(item => item.method === 'POST').length, evaluationCount)
  await click([...host.querySelectorAll('[role="tab"]')].find(item => item.textContent === '单次评估'))
  assert.equal(keyInput().value, 'my-custom-key'); assert.equal(host.querySelector('textarea').value, '{bad')
  assert.ok(host.querySelector('textarea').getAttribute('aria-invalid') === 'true')
  assert.ok(host.textContent.includes('最终值 · my-custom-key'))
  await act(async () => root.unmount()); root = undefined; host.remove()
  await mount(<App />)
  assert.equal(keyInput().value, 'checkout-redesign-2026'); assert.equal(host.querySelector('[role="tab"][aria-selected="true"]').textContent, '单次评估')
  assert.ok(!host.textContent.includes('最终值 · my-custom-key')); assert.ok(host.querySelector('textarea').value.includes('"country": "CN"'))
})

test('editable key candidates load every page at size 100, show name/status, search and select disabled key', async () => {
  network(({ url }) => response(Number(url.searchParams.get('page')) === 2 ? page([flag('last-key', 'production', false)], 101, 2) : page(Array.from({ length: 100 }, (_, i) => flag(`flag-${i}`)), 101)))
  await mount(inputs()); await focusKey()
  await waitFor(() => requests.some(item => item.url.searchParams.get('page') === '2'), 'second candidate page requested')
  assert.ok(requests.every(item => item.url.searchParams.get('page_size') === '100' && item.url.searchParams.get('environment') === 'production' && !item.url.searchParams.has('enabled')))
  await input(keyInput(), 'last')
  await waitFor(() => document.body.textContent.includes('名称 last-key'), 'last page option visible')
  const option = [...document.querySelectorAll('.ant-select-item-option')].find(item => item.textContent.includes('last-key'))
  assert.ok(option.textContent.includes('已停用'))
  await click(option); assert.equal(keyInput().value, 'last-key')
  await input(keyInput(), 'never-created'); assert.equal(keyInput().value, 'never-created')
})

test('environment changes retain custom key and ignore late candidates from the previous environment', async () => {
  let resolveProduction
  network(({ url }) => url.searchParams.get('environment') === 'production' ? new Promise(resolve => { resolveProduction = resolve }) : response(page([flag('staging-key', 'staging')])))
  await mount(inputs()); await focusKey()
  await waitFor(() => Boolean(resolveProduction), 'production request started')
  await chooseEnvironment('staging')
  assert.equal(keyInput().value, 'custom-key')
  await input(keyInput(), ''); await focusKey()
  await waitFor(() => document.body.textContent.includes('staging-key'), 'staging candidates visible')
  await act(async () => resolveProduction(response(page([flag('stale-key')]))))
  assert.ok(!document.body.textContent.includes('stale-key'))
  assert.ok(requests.find(item => item.url.searchParams.get('environment') === 'production').signal.aborted)
})

test('candidate feedback supports loading, error, retry, empty and refresh after refocus', async () => {
  let resolveRequest
  network(() => new Promise(resolve => { resolveRequest = resolve }))
  await mount(inputs()); await focusKey()
  assert.ok(document.body.textContent.includes('正在加载'))
  await act(async () => resolveRequest(response({ error: { code: 'INTERNAL_ERROR', message: 'failure' } }, 500)))
  await waitFor(() => document.body.textContent.includes('候选加载失败'), 'inline failure')
  assert.equal(keyInput().value, 'custom-key')
  await click(button('重试'))
  await act(async () => resolveRequest(response(page())))
  await waitFor(() => document.body.textContent.includes('暂无 Flag'), 'empty feedback')
  const count = requests.length
  await act(async () => keyInput().blur()); await settle(); await focusKey()
  assert.ok(requests.length > count)
  await act(async () => resolveRequest(response(page([flag('fresh-key')]))))
  await input(keyInput(), 'fresh')
  await waitFor(() => document.body.textContent.includes('fresh-key'), 'fresh option visible')
})

test('delete confirmation identifies key/environment and permanent cascade; cancel sends no DELETE', async () => {
  network(() => response(page([flag('delete-key')], 1, 1, 20)))
  await mount(<MemoryRouter><FlagListPage /></MemoryRouter>)
  await waitFor(() => button('删除', host), 'delete action visible')
  await click(button('删除', host))
  await waitFor(() => document.querySelector('[role="dialog"]'), 'confirmation open')
  const dialog = document.querySelector('[role="dialog"]')
  assert.ok(dialog.textContent.includes('delete-key') && dialog.textContent.includes('production'))
  assert.ok(dialog.textContent.includes('相关规则与操作历史一并删除，无法撤销'))
  await click(button('取消', dialog)); assert.equal(requests.filter(item => item.method === 'DELETE').length, 0)
})

test('failed deletion preserves the row; pending deletion guards competing status and repeated delete; success reloads', async () => {
  let resolveDelete, removed = false
  network(({ method }) => method === 'DELETE' ? new Promise(resolve => { resolveDelete = resolve }) : response(page(removed ? [] : [flag('delete-key')], removed ? 0 : 1, 1, 20)))
  window.history.replaceState({}, '', '/')
  await mount(<App />)
  await waitFor(() => button('删除', host), 'delete action visible')
  await click(button('删除', host)); await click(button('删除', document.querySelector('[role="dialog"]')))
  await waitFor(() => Boolean(resolveDelete), 'delete started')
  assert.ok(host.querySelector('[role="switch"]').disabled); assert.ok(button('删除', host).disabled)
  await click(host.querySelector('[role="switch"]')); await click(button('删除', host))
  assert.equal(requests.filter(item => item.method === 'DELETE').length, 1); assert.equal(requests.filter(item => item.method === 'PATCH').length, 0)
  await act(async () => resolveDelete(response({ error: { code: 'INTERNAL_ERROR', message: '删除失败' } }, 500)))
  await waitFor(() => !button('删除', host).disabled, 'failed delete unblocks row')
  assert.ok(host.textContent.includes('delete-key'))
  assert.ok(document.body.textContent.includes('删除失败'), 'API failure is visible to the user')
  // Close a failed confirmation if the component deliberately leaves it available for retry.
  if (document.querySelector('[role="dialog"]')) await click(button('取消', document.querySelector('[role="dialog"]')))
  await click(button('删除', host)); await click(button('删除', document.querySelector('[role="dialog"]')))
  removed = true; await act(async () => resolveDelete(response(null, 204)))
  await waitFor(() => !host.textContent.includes('delete-key'), 'successful delete reloads row')
})

test('deleting last row clamps page while retaining filters; pending status prevents delete', async () => {
  let removed = false, resolveStatus
  network(({ url, method }) => {
    if (method === 'PATCH') return new Promise(resolve => { resolveStatus = resolve })
    if (method === 'DELETE') { removed = true; return response(null, 204) }
    const current = Number(url.searchParams.get('page'))
    return response(page(current === 2 && !removed ? [flag('last-row')] : current === 2 ? [] : Array.from({ length: 20 }, (_, i) => flag(`first-${i}`)), removed ? 20 : 21, current, 20))
  })
  await mount(<MemoryRouter><FlagListPage /></MemoryRouter>)
  await chooseOption(host.querySelectorAll('.filter-bar .ant-select')[0], 'production')
  await chooseOption(host.querySelectorAll('.filter-bar .ant-select')[1], '已启用')
  await input(host.querySelector('input[aria-label="搜索名称或 key"]'), 'retained-filter'); await settle(350)
  await waitFor(() => host.querySelector('.ant-pagination-item-2'), 'list page 2 available')
  await click(host.querySelector('.ant-pagination-item-2'))
  await waitFor(() => host.textContent.includes('last-row'), 'page 2 row visible')
  await click(host.querySelector('[role="switch"]'))
  assert.ok(button('删除', host), 'delete action exists')
  assert.ok(button('删除', host).disabled)
  await act(async () => resolveStatus(response({ ...flag('last-row'), enabled: false })))
  await waitFor(() => !button('删除', host).disabled, 'status completed')
  await click(button('删除', host)); await click(button('删除', document.querySelector('[role="dialog"]')))
  await waitFor(() => host.querySelector('.ant-pagination-item-1.ant-pagination-item-active') && !host.textContent.includes('last-row'), 'page clamps to 1')
  const lastGet = requests.filter(item => item.method === 'GET').at(-1)
  assert.equal(lastGet.url.searchParams.get('q'), 'retained-filter'); assert.equal(lastGet.url.searchParams.get('page'), '1')
  assert.equal(lastGet.url.searchParams.get('environment'), 'production'); assert.equal(lastGet.url.searchParams.get('enabled'), 'true')
})

test('MSW mirrors DELETE UUID validation, 204 cascade, 404 and environment isolation', async () => {
  assert.ok(handlers.some(handler => handler.info.header === 'DELETE /api/v1/flags/:id'), 'MSW exposes whole-Flag DELETE handler')
  const id = '90000000-0000-4000-8000-000000000001'
  const siblingId = '90000000-0000-4000-8000-000000000002'
  flags.push({ ...flag('mock-deletion'), id }, { ...flag('mock-deletion', 'staging'), id: siblingId })
  rules.push({ id: 'mock-rule', flag_id: id, priority: 1, attribute: 'x', operator: 'equals', expected_value: 1, roll_value: true })
  histories.set(id, [{ id: 1, action: 'flag_created' }])
  globalThis.fetch = nativeFetch
  const server = setupServer(...handlers)
  server.listen({ onUnhandledRequest: 'error' })
  try {
    assert.equal((await fetch('http://localhost/api/v1/flags/invalid', { method: 'DELETE' })).status, 400)
    assert.equal((await fetch(`http://localhost/api/v1/flags/${id}`, { method: 'DELETE' })).status, 204)
    assert.ok(!flags.some(item => item.id === id)); assert.ok(!rules.some(item => item.flag_id === id)); assert.ok(!histories.has(id))
    assert.ok(flags.some(item => item.id === siblingId))
    const again = await fetch(`http://localhost/api/v1/flags/${id}`, { method: 'DELETE' })
    assert.equal(again.status, 404); assert.equal((await again.json()).error.code, 'FLAG_NOT_FOUND')
  } finally {
    server.close()
    for (const target of [id, siblingId]) { const index = flags.findIndex(item => item.id === target); if (index >= 0) flags.splice(index, 1); histories.delete(target) }
    const index = rules.findIndex(item => item.flag_id === id); if (index >= 0) rules.splice(index, 1)
  }
})

test('fresh focus exposes existing choices while preserving a custom key; unmatched search explains custom input', async () => {
  network(() => response(page([flag('existing-key')])) )
  await mount(inputs()); await focusKey()
  await waitFor(() => [...document.querySelectorAll('.ant-select-item-option')].some(item => item.textContent.includes('existing-key')), 'focus exposes candidates despite custom form key')
  assert.equal(keyInput().value, 'custom-key')
  await input(keyInput(), 'no-such-key')
  await waitFor(() => document.body.textContent.includes('没有匹配的 Flag'), 'no-match explains custom input is accepted')
})

test('console retains a server evaluation error through route navigation without resubmitting', async () => {
  network(({ url }) => url.pathname === '/api/v1/evaluate' ? response({ error: { code: 'FLAG_NOT_FOUND', message: 'missing' } }, 404) : response(page([], 0, 1, Number(url.searchParams.get('page_size') ?? 20))))
  await mount(<App />)
  await click(button('开始评估'))
  await waitFor(() => host.textContent.includes('Flag 不存在：检查 key 与环境'), 'server evaluation error rendered')
  await click([...host.querySelectorAll('a')].find(item => item.textContent === 'Flag 列表'))
  await click([...host.querySelectorAll('a')].find(item => item.textContent === '评估控制台'))
  assert.ok(host.textContent.includes('Flag 不存在：检查 key 与环境'))
  assert.equal(requests.filter(item => item.method === 'POST').length, 1)
})
