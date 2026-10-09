# 任务 T4b：前端详情 / 规则 / 历史 / 评估控制台

## 你的角色

你是 FlagForge 的前端工程师（第二阶段），负责本项目**用户体验的精华**：
Flag 详情页（规则管理 + 历史 Diff）、评估控制台（单次评估 + Trace 轨迹 + 批量模拟分布）。
评估控制台是评委第一眼看的页面，信息层级优先于装饰。

## 必读（动手前读完）

1. `docs/plan/01-design.md` §6.2~§6.5（契约）、§8-A/B/C/D（四个亮点规格）、§9
2. `docs/plan/02-edge-cases.md` 组 3、组 5、组 6、组 7、组 8 的前端行为 + 组 9
3. T4a 报告（client/mocks/布局现状与约定）

## 交付物

### 1. FlagDetailPage `/flags/:id`

**基本信息卡**：全字段 + 状态 Tag + 「编辑」入口；（stretch）环境复制：目标环境 Select +
确认弹窗（明示「复制后新 flag 为停用状态」）→ 409 时 message 提示目标环境已有同 key。

**规则 Tab**（组 3）：
- 表格按 priority 升序：`#1` 徽标、attribute、operator、expected_value（紧凑 JSON 等宽展示）、
  `→ true/false` 结果列（绿/灰 Tag）、操作（编辑/删除）
- 新建/编辑规则 Modal：
  - attribute 输入框；operator Radio（equals/in）；roll_value Switch 或 Radio
  - priority InputNumber（≥1）；失焦/提交时 409 → priority 字段内联报错「该优先级已被占用」（组 9.2）
  - **expected_value 编辑器**：JSON 文本域 + 即时校验 + 格式化按钮 + 按操作符的提示文案
    （equals：「JSON 标量，如 `"pro"` / `42` / `true`」；in：「非空标量数组，如 `["US","CA"]`）。
    不做智能类型推断控件 —— 显式 JSON 最不易错（设计口径 D3 的 UI 侧表达）
  - 提交前本地校验标量/非空数组规则，与后端口径一致（组 3.4~3.7 前端拦截）
- 删除 → Popconfirm（组 9.6）

**历史 Tab**（组 7，亮点 C）：
- 时间线（AntD Timeline）：action 图标区分（创建/更新/启停/规则），operator、created_at 相对时间
- `DiffView` 组件：读 summary.changes 逐字段渲染 `old → new`（old 删除线红、new 绿；
  布尔值渲染为 Tag）；rule_* 操作渲染规则摘要行（如 `#2 plan equals "pro" → true`）；
  flag_created / flag_copied 渲染对象摘要卡
- 分页加载更多

### 2. ConsolePage `/console`（亮点 A + B 主战场）

**Tab 1 单次评估**（组 5）：
- 输入区：flag_key 输入框、environment Select、user_attributes JSON 文本域
  （`{ "country": "CN", "plan": "pro" }` 占位示例）+ 「格式化」按钮 + 即时校验
  （语法错不发请求、标行提示，组 9.3）
- 结果卡：
  - **最终值大字**：`true` 绿 / `false` 灰，字号显著大于其他元素
  - reason 徽标：`rule_hit`（命中 #2 · plan equals "pro"）/ `default`（未命中规则，返回默认值）/
    `disabled`（Flag 已停用，短路返回 false）
  - **TraceTable**（亮点 A）：每行 priority/attribute/operator/matched/cause；
    matched=true 行高亮；cause → 中文文案常量表（`constants/causeText.ts`）：
    matched=命中 / attribute_missing=属性缺失 / value_not_equal=值不相等 /
    value_not_in_list=不在列表中 / type_mismatch=类型不匹配 / short_circuited_by_disabled_flag=停用短路
  - 「复制 curl」按钮：生成当前评估的 curl 命令入剪贴板
- 错误分支：404 → 「Flag 不存在：检查 key 与环境」；400 INVALID_JSON/INVALID_USER_ATTRIBUTES → 表单内联

**Tab 2 批量模拟**（组 6，亮点 B）：
- 引导文案：「上线前模拟：不同用户群分别拿到什么？每行一个 JSON」
- textarea：每行一个 JSON object；本地校验（空行忽略、坏行标行号，组 6.6）；行数计数与 100 上限提示
- 结果区：
  - 统计卡三枚：true 数 / false 数 / 默认值比例
  - 「命中规则分布」列表：规则摘要 + 命中次数 + 横条占比（按次数降序）
  - 明细表：index、value Tag、reason 摘要；行可展开看该组 trace（复用 TraceTable）

### 3. MSW handlers 扩充

- 规则 CRUD（含 409 优先级）、evaluate 全分支（命中/默认/停用/404/400）、batch（含统计）、copy
- mock 数据讲 seed 的故事（01-design §11），让前端演示不依赖后端也有真实感

## 本任务已知陷阱

- 路由 /flags/:id 的 Tab 状态：切换 Tab 不重拉详情；规则/历史数据各自懒加载
- 409 DUPLICATE_PRIORITY 的内联定位需要 API 返回后手动 `form.setFields`（AntD 不会自动映射）
- 批量 textarea 的行号报告与后端 index（0 起）口径：前端展示用 1 起行号，映射关系写注释
- cause 常量表必须覆盖 §6.3 全部 6 个枚举，漏一个就渲染裸英文
- 复制 curl：注意 shell 引号转义（JSON 里的双引号）

## 验收标准（对 MSW 生效；若后端已就绪，双重验证加分）

- [ ] 详情页三块信息完整；规则 CRUD 全流程 + 409 内联 + 删除确认
- [ ] 历史 Tab：8 种 action 的 mock 数据全部渲染正确（DiffView 的 old→new 视觉正确）
- [ ] 控制台：seed 场景「country=CN plan=pro 评估 production checkout-redesign-2026」
      → 命中 #2 规则 → true，trace 两行（#1 不在列表 / #2 命中）—— 这是评委的黄金演示路径
- [ ] 停用 flag 评估 → false + disabled 徽标 + 短路 trace
- [ ] 批量：5 组混合数据统计卡与分布列表数字正确
- [ ] 每个数据区域有加载态；列表空态不白屏；`npm run build` 零 TS 错误

## 硬约束

- 不改 T4a 的 client/布局约定；需要新端点函数在 client.ts 追加并同步 MSW
- stretch（环境复制 UI）在核心全验收后再做；时间紧按 00-README 裁剪顺序砍
- 组件 ≤300 行；TraceTable/DiffView 是独立可复用组件

## 完成后必做

写 `docs/ai-logs/T4b-report.md`（模板同 T0；「关键决策」包含：expected_value 编辑器取舍、
批量行号口径、黄金演示路径截图或描述），然后
`git commit -m "feat(T4b): flag detail with rules/history, evaluation console with trace and batch"`。

## 调度补充（完整归档）

你负责 FlagForge T4b，用户已授权继续完成完整题目未完成部分。只在 C:/Users/77230/FlagForge/.worktrees/t4b 工作，分支 codex/t4b-detail-console，基线1769032，PowerShell、Git Bash C:/Program Files/Git/bin/bash.exe。阅读 docs/plan/prompts/T4b-frontend-detail-console.md 严格执行全部步骤，并读design §6.2~6.5/§8/§9、edge组3/5/6/7/8/9、T4a-report。现有设计已批准，无需新脑暴或用户确认。
关键要求：实现FlagDetailPage基本信息全字段/状态/编辑入口，规则Tab按priority升序并清楚说明数字越小越优先、重复拒绝。规则新建编辑Modal属性/operator/priority整数>=1/expected JSON即时验证及格式化/roll_value；equals仅string|number|bool，in非空全标量数组；409 DUPLICATE_PRIORITY字段内联，删除Popconfirm。历史Tab懒加载/分页更多/Timeline8种action/独立DiffView老值红删除线新值绿及booleanTag，rule摘要和flag对象卡按设计summary。
Console单次flag_key/env/user_attributes JSONobject输入，语法本地拦截及标行、格式化；结果value大字reason hit/default/disabled、独立TraceTable6cause全中文、高亮命中行，黄金seed countryCN planpro→#2true trace2行；停用billing→false短路。404友好可修正，400内联；复制curl正确shell单引号转义。批量每行JSONobject、忽略空行、错误1起原始行号（后端index0起），<=100，统计/分布/明细可展开trace。
扩展client追加rules/evaluate/batch函数及契约类型、扩展MSW规则CRUD/评估/批量全分支；mock前四个flag和规则应真实讲seed故事，不能全部plan=pro。禁止改变已有T4a客户端错误/取消约定、原路由布局、已有列表/编辑行为。核心验收完可再做环境复制UI/client/mock，属于stretch，不要牺牲必做项。所有组件<=300行，可拆小组件。每数据区loading/空态/失败可重试、防双提交。保持依赖栈，不新增前端测试框架（设计§10明确不做），通过临时内存实际handler/解析函数红绿检查和真实浏览器验收，不写镜像实现测试。
并行T3a只改backend，无共享源码冲突。main端口5173/8787正在运行，不可停止；自己dev用5174或5175（先检查），MSW mock，本root最后真实联调。npm ci可用国内源，Node/npm已安装。不要改README/scripts/backend，不要改默认mock开关（留给T5），不要启动Compose/数据库。
完成归档全文 T4b-prompt.md、T4b-report.md（选择/行号/黄金演示截图或描述/真实测试证据与限制），提示词commit；不要创建下级agents，root独立审查。回报状态、SHA、变更、build/lint/浏览器验收、未尽问题，停止自有临时服务、保留截图文件，不影响root服务。

### 追加指导

补充：本轮完整题面验证明确要求UI与README写明优先级方向/拒绝重复策略，规则Tab请直接展示。api契约准确请求字段为 flag_key、user_attributes，批量user_attributes_list，勿用上轮验收探测中仅示意的 key/user_attrs。root将安排真实Go联调及83条边界验收，截图请存frontend/evidence/T4b-*.png。

用户已明确要求继续任务，请从保留的 .worktrees/t4b 未提交进度恢复，继续你未完成的浏览器验证、针对性修复、build/lint、T4b-prompt/report与commit，发SHA供root审查。不要重新实现。root看到5174由你的PID100使用，结束时停止自己服务并保留截图。若额度仍不可用，保留文件即可，root会接手；不要反复重试。

核心自验已有实测证据，可按原计划stretch补环境复制UI（目标Select、确认明示新flag停用、同源400/冲突409、成功跳新详情），沿用已完成handler/client，不扩展其他选项。后端T3b会在核心全绿后尝试copy。请完成后独立验证并尽快报告/commit，root不以复制阻塞必做项交付。

### root有限数值复审指导（完整）

root读到 detail/console已实现，public/mockServiceWorker.js diff为空（可能纯CRLF工作区标记），提交时不要人为改生成内容。质量审查请留意parseJSON的非有限数字：JSON.parse('1e400')得到Infinity，JSON.stringify会变null；expected editor和user_attributes本地校验应明确拒绝不可JSON序列化数字，避免静默改输入语义。可先定向红绿函数检查再修；不是要求添加框架。
