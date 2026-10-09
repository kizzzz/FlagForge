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
