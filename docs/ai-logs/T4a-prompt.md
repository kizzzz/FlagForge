# T4a 提示词归档

执行日期：2026-10-09（Asia/Shanghai）
来源：docs/plan/prompts/T4a-frontend-base.md，以下原始任务全文。

# 任务 T4a：前端骨架 + Flag 列表 + 新建/编辑

## 你的角色

你是 FlagForge 的前端工程师（第一阶段）：搭建前端基建（路由/布局/API client/MSW mock），
并实现 Flag 列表页与新建/编辑页。**开发期全程用 MSW mock，不依赖后端进度**；
但 API 层按契约封装，后端就绪后关掉 mock 即通。

## 必读（动手前读完）

1. `docs/plan/01-design.md` §2（栈）、§6 前言+§6.1+§6.6（契约+错误信封）、§9（页面质量要求）
2. `docs/plan/02-edge-cases.md` 组 1、组 2、组 9
3. T0 报告（脚手架现状、选了 fetch 还是 axios）

## 交付物

### 1. 基建（会被 T4b 复用，稳定优先）

- `src/api/client.ts`：按 §6.1 契约封装全部 flag 端点函数；统一解析错误信封
  （非 2xx → 抛 `ApiError{code,message,details}`）；分页类型 `Paged<T>`
- `src/mocks/handlers.ts` + `src/mocks/browser.ts`：MSW handlers 覆盖已封装端点，
  **含错误分支**（同 key 创建 → 409 DUPLICATE_FLAG_KEY 等，供联调与自测）
- MSW 开关：`VITE_USE_MSW` 环境变量（`.env.development` 默认 true，`.env.production` false），
  main.tsx 按 flag 条件启动 worker —— T5 集成时改一行即切真实后端
- 路由：`/`（列表）、`/flags/new`、`/flags/:id/edit`、`/flags/:id`、`/console`（占位页「T4b 实现」）
- 布局：顶部导航（Flag 列表 / 评估控制台）+ AntD ConfigProvider 主题
  （紧凑为主，主色避开默认蓝可选，信息层级清晰即可，不追求精美）
- 全局 fetch 层错误 → AntD message 反馈的统一拦截

### 2. FlagListPage（组 2）

- 筛选：environment Select（三枚举+全部）、enabled Select、搜索框（name/key，防抖 300ms）
- 表格列：name、key（等宽字体 + 一键复制）、environment Tag（三色区分）、enabled **Switch**
  （乐观更新：先翻转 UI，请求失败回滚 + 错误 message，组 2.8）、default_value、rule_count、
  updated_at（dayjs 本地化格式）、操作（详情/编辑）
- 服务端分页（AntD Table 受控 pagination）；空态：引导文案 + 「新建 Flag」按钮
- 加载态：Table loading；筛选变更重置到第 1 页

### 3. FlagEditPage（新建/编辑共用，组 1）

- 新建：name、key、environment、default_value（Radio true/false）、description
  - key 即时校验：正则 `^[a-z][a-z0-9._-]{1,63}$` 与后端逐字符一致（组 9.7），
    校验文案给出格式说明与示例
- 编辑：**key 与 environment 只读**（灰显 + tooltip 说明「key 是调用方锚点，创建后不可变，设计决策 D6」）；
  仅可改 name / description / default_value
- 提交：loading 禁用防重复点击（组 9.1）；成功 → message + 跳详情页
- **409 内联**：ApiError.code === DUPLICATE_FLAG_KEY → form.setFields 定位到 key 字段
  显示「该 key 在此环境已存在」；其余错误走全局 message（组 9.2）

## 本任务已知陷阱

- MSW 2.x 的 handler 语法（`http.post` 不是 `rest.post`）与 vite 插件配置；
  worker 文件需 `npx msw init public/`
- AntD Form 受控语义：`initialValues` + `form.setFieldsValue` 用于编辑态数据回填，
  路由参数变化（/flags/a/edit → /flags/b/edit）时需手动重置
- 错误信封解析放在 client.ts 一处，页面不重复 parse response
- 端口：dev 5173，`/api` 走 vite proxy（T0 已配）；MSW 开启时 interceptor 优先于 proxy，
  不会产生真实请求 —— 自测时开 Network 面板确认

## 验收标准（对 MSW mock 生效）

- [ ] 列表：筛选/搜索/分页/空态/加载态全部可用（组 2.3/2.5/9.4）
- [ ] Switch 乐观更新：mock 一个失败响应，验证回滚 + message（组 2.8）
- [ ] 新建：key 正则拦截非法输入（大写/空格/数字开头，组 9.7）；合法创建成功跳转
- [ ] 新建撞 key：mock 409 → key 字段内联报错（组 9.2）
- [ ] 编辑态 key/env 只读且有说明 tooltip
- [ ] `npm run build` 零 TS 错误

## 硬约束

- 不实现详情页/控制台/批量（T4b 范围），路由占位即可
- 组件文件 ≤300 行，超出拆子组件（后续任务要在同一代码库上工作）
- 不引入额外 UI 库；状态管理用 React 内建（本项目规模不需要 redux/zustand）

## 完成后必做

写 `docs/ai-logs/T4a-report.md`（模板同 T0；「关键决策」包含：MSW 开关机制、
API client 的错误解析设计、页面文件结构图），然后
`git commit -m "feat(T4a): frontend foundation, flag list and edit pages"`。


## 本次调度补充（完整约束归档）

- 波2 T0 → T1 ∥ T2 ∥ T4a；T0 已通过全部验收，基线提交 9a3bf66。
- 独立 worktree C:/Users/77230/FlagForge/.worktrees/t4a；分支 codex/t4a-frontend-base。
- 每条 shell 命令显式 workdir；只修改此 worktree，禁止修改主 checkout、切分支、merge、cherry-pick、创建其他 agent。
- 仅修改 frontend、T4a-report.md 与 T4a-prompt.md；不改 backend、根 README 或其他任务报告。
- 设计与实施/提交已获授权；不重新设计，不重新请求批准。
- API 错误集中转换为 ApiError，全局 message 不与页面重复提示；Paged<T> 共享。
- MSW2 http handlers 覆盖 §6.1 全部 Flag 端点及错误分支，分页、AND 筛选和大小写不敏感搜索遵守契约。
- 提交 .env.development=true、.env.production=false；开关关闭即真实 fetch。
- 快速筛选须防请求竞态；Switch 单行 pending、失败只回滚对应行，不影响其他行。
- 编辑路由 id 变化须 resetFields + setFieldsValue；loading 禁提交；409 DUPLICATE_FLAG_KEY 内联到 key。
- 组件 ≤300 行，React 内建状态，不新增 UI 库与前端测试框架；设计 §10 前端不写测试。
- 必须实际浏览器验证验收行为；先读 computer-use skill，通过 cua_repl 操作自己的临时 tab，不能绕过工具做 UI 自动化。
- 不支持的浏览器验证必须如实记录，不能勾为已通过。
- 可复制主 checkout frontend/node_modules 到本 worktree，不共享可变源码、不修改主依赖。
- Node 24.13、npm 11.6、React 18.3.1、AntD5、router6、MSW2 已锁定；5173 本波只供 T4a。
- 结束释放服务，或向调度者报告 PID/session；完成返回 commit SHA 与验证证据。
- 提交消息：feat(T4a): frontend foundation, flag list and edit pages。
