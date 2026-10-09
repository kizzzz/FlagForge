# T4a 执行报告

执行日期：2026-10-09（Asia/Shanghai）
worktree：C:/Users/77230/FlagForge/.worktrees/t4a
分支：codex/t4a-frontend-base
基线：9a3bf66（T0）
状态：T4a 核心验收完成，build/lint 通过。浏览器附加验证的限制见「未尽事项」。

## 做了什么

- 阅读 01-design §2、§6（含 §6.1 / §6.6）、§9，02-edge-cases 组1/2/9、T0-report、T4a 原始任务；归档完整原始提示词及调度补充。
- 原生 fetch client 封装 §6.1 六个端点：list / get / create / update / setStatus / history；共享 Flag、Rule、History、Paged<T>、ApiError 类型。
- MSW2 handlers 实现六端点、26 条演示数据（足够验证分页）与详情规则/历史数据；覆盖重复环境+key 409、非法 JSON/key/环境、字段校验、不可变字段、404、分页参数错误和一次可复现的状态 500。
- 运行 npx msw init public/ --save，提交生成的 public/mockServiceWorker.js；只忽略此生成文件的 ESLint 检查，不改其源码。
- 实现顶部导航、AntD ConfigProvider/中文 locale/绿色主题、五个指定路由及未知路由 404。详情与控制台均仅显示 T4b 实现。
- 列表实现 AND 筛选、300ms 搜索防抖、受控服务端分页/页大小、空态新建引导、loading/失败重试、Tag、等宽 key 与 AntD copyable、dayjs 本地时间、独立行内启停、成功提示。
- 新建/编辑共用表单：name 空白与长度校验、key 即时正则、默认值 radio、说明；编辑 key/env 灰显只读及 D6 tooltip（支持 hover/focus），只 PATCH 可编辑字段；409 只定位 key；提交防重复；成功跳详情。
- 未引入额外 UI 库、状态库、前端测试框架或自动化测试文件。

## 关键决策与原因

### MSW 开关机制

.env.development 的 VITE_USE_MSW=true；.env.production=false，两文件提交且不被忽略。main.tsx 先按字符串 true 动态 import/start worker，完成后再渲染 React。关闭开关后 fetch URL 与页面完全相同，直接走真实 /api/v1，Vite proxy 仍使用 T0 的 8787 配置。生产构建已排除 mock 浏览器模块。

Mock 在模块内存中维护 flags/rules/history，刷新或 handler 热重载会重置演示数据。billing-portal 的首次 /status 变更固定返回 500 INTERNAL_ERROR，后续重试成功；仅存在于 mock，不污染真实 API。列表 450ms、表单 600ms、状态 900ms 延迟便于观察加载与乐观更新。

### API 错误设计

request 在一处读取 JSON / 非 2xx error 信封，转换为 ApiError(code, message, details, status)。网络异常统一转 NETWORK_ERROR；非 JSON 的非 2xx 提供 HTTP 状态与重试提示。Shell 通过 AntD App.useApp 的 message 注册全局 reporter，页面不重复解析或错误 toast。创建调用显式 inlineErrors=[DUPLICATE_FLAG_KEY]，由表单唯一显示内联错误；其余错误统一 reporter。AbortError 不提示。

列表 effect 为每次筛选/分页使用 AbortController，废弃旧请求并检查 aborted，避免旧结果覆盖新筛选。启停以 id 保存 pending，UI 立即更新并禁用该 Switch；失败用函数式 setState 只恢复该 id，成功重新请求当前分页。并行列表响应保留 pending 的乐观值。表单路由变化 resetFields/setFieldsValue，取消旧详情请求；旧路由提交响应不再触发当前路由状态或跳转。

### 页面文件结构

```text
frontend/src/
├── App.tsx                 # ConfigProvider、message 接入、布局与路由
├── main.tsx                # 可切换 MSW bootstrap
├── api/client.ts           # 契约类型 + 六端点 + 错误统一处理
├── mocks/
│   ├── browser.ts          # setupWorker
│   ├── handlers.ts         # MSW2 handlers/校验/分页
│   └── data.ts             # 演示 flags/rules/history
└── pages/
    ├── FlagListPage/index.tsx  # 82 行
    └── FlagEditPage/index.tsx  # 81 行
```

App.tsx 42 行，main.tsx 13 行；所有组件均小于 300 行。

## 验证命令与输出摘要

命令均从本 worktree 的 frontend 执行（Git/文件检查从本 worktree 根执行）。

| 验证 | 结果 |
| --- | --- |
| npx msw init public/ --save | exit 0，worker successfully copied，package.json workerDirectory=public |
| npm run build | exit 0，TypeScript 无错误，Vite 1475 modules transformed，built in 11.20s |
| npm run lint | exit 0，0 errors / 0 warnings |
| git diff --check | exit 0，无空白错误（Git 仅提示本机 LF/CRLF 转换） |
| TSX 行数检查 | 最大 82 行，满足 ≤300 |
| 结束端口检查 | Get-NetTCPConnection 查询 5173/5174 无监听 |

Vite 生产包约 1.07 MB（gzip 340.7 KB）存在 >500 KB chunk 提示，不是 TS/lint 错误；未扩大 T4a 范围做代码分包优化。

## 实际浏览器自验（cua_repl，独立临时 tab）

| 项目 | 实际证据与结果 |
| --- | --- |
| MSW 运行 | dev 日志 [MSW] Mocking enabled；GET /flags 200；PATCH status 500 由 MSW 命中 |
| 列表分页/加载 | 初始26条、第一页20条；点击下一页显示21–26；请求中 .ant-spin-spinning 可见 |
| 筛选回第一页 | 第二页选择 development，上一页按钮 disabled，回到第一页 |
| 搜索与组合筛选 | CHECKOUT 大写命中 key；production + CHECKOUT 得1条；再选已停用得0条，出现空态与新建引导；BILLING 大写命中 billing-portal |
| Switch pending/失败回滚 | 点击 billing-portal 时 aria-checked=true 且 disabled；500 后 aria-checked=false，错误提示文本仅1条；重试成功 true 与 Flag 已启用 message |
| key 即时校验 | Bad-Key / bad key / 1bad 输入各产生 aria-invalid=true 和格式文案；提交仍在 /flags/new |
| 409 内联 | development + checkout-redesign-2026 返回409；key 下方显示该 key 在此环境已存在；提交时按钮 disabled |
| 合法创建 | a.b-c_d 与 t4a-browser-proof 创建成功；跳 /flags/<uuid>，出现 Flag 已创建；列表可见新记录及默认值 |
| 编辑 | 新记录加载期间字段/提交禁用；回填后 key/env disabled；tooltip 文案含调用方锚点与 D6；改 name 保存后跳详情且 Flag 已保存 |
| 占位范围 | 详情显示 Flag 详情 + T4b 实现；控制台显示评估控制台 + T4b 实现 |
| 关闭 mock | 生产预览5174无 MSW 日志；Vite proxy 实际记录 /api/v1/flags ECONNREFUSED（8787未启动） |
| 真实请求错误恢复 | 列表 HTTP500 显示统一错误message、失败Alert及重试按钮；点击重试再次请求；创建失败保持表单输入与当前路由 |

截图归档于 frontend/evidence：T4a-list.jpg、T4a-duplicate-key.jpg、T4a-edit-readonly.jpg、T4a-switch-rollback.jpg、T4a-created.jpg、T4a-real-api-error.jpg。

## 未尽事项 / 需要人工决策

- T4a 指定的六项核心验收均实际通过，无待人工决策。
- 额外的复制功能使用 AntD Typography.copyable，已实际点击；当前隐藏浏览器的 clipboard.readText 返回空串，不能据此确认操作系统剪贴板内容，因此未声称剪贴板写入已完整验收。T5 可在可见浏览器手动复制/粘贴确认。
- 浏览器工具未提供 DevTools Network 面板；未伪称已打开该面板。拦截机制通过 MSW 捕获日志，以及关闭 mock 时真实 Vite proxy ECONNREFUSED 日志分别验证。
- 真实后端成功联调属于 T5；本次未启动/依赖后端。详情、规则编辑、评估控制台、批量模拟仍留给 T4b。
- React Router6 仅输出 v7 future flags 迁移提示，无业务运行时错误；未因未升级的未来版本扩大范围。
- 临时浏览器 tab1/tab2 已关闭；dev session35083 与 preview session30976 已 Ctrl+C 结束，5173/5174 已释放；主 checkout 和其他任务文件未修改。

## 验收自验清单

- [x] 列表筛选/搜索/分页/空态/加载可用。
- [x] mock Switch 失败→乐观回滚+唯一错误message。
- [x] key 正则拦截非法输入；合法创建并跳详情。
- [x] 同环境重复key409→字段内联。
- [x] 编辑态key/environment只读与D6 tooltip。
- [x] npm run build零TS错误；npm run lint通过。
