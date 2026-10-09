# FlagForge 设计文档 v1.0

> 定稿日期：2026-10-09。本文档是实现的唯一契约源。executor 与本文冲突时，以本文为准；
> 发现本文漏洞 → 在执行报告中上报，不要擅自改设计。

## 1. 产品定位

**FlagForge** —— 面向小团队的 Feature Flag 与灰度规则平台：按环境（development / staging / production）
管理功能开关，为每个 Flag 配置有序的用户属性匹配规则，并提供带**评估轨迹（Trace）**的在线评估控制台。

核心体验主张：**不只告诉你结果，还告诉你为什么**（每条规则为何命中/为何跳过）。

## 2. 技术栈（定稿，不再讨论）

| 层 | 选型 |
|----|------|
| 后端 | Go 1.22+ / Gin / GORM（仅作数据访问，禁用 AutoMigrate）/ golang-migrate（纯 SQL 迁移） |
| 数据库 | PostgreSQL 16（Docker Compose，README 附本地实例备选） |
| 后端测试 | testify + table-driven；评估引擎为纯函数单测（不碰 DB） |
| 前端 | Vite + React 18 + TypeScript + Ant Design 5 + react-router v6 + dayjs |
| 前端开发期 mock | MSW（Mock Service Worker），按本文契约写 handler |
| 端口 | 后端 `:8787`，前端 `:5173`（Vite dev proxy `/api` → `8787`，免 CORS） |

## 3. 仓库结构

```
FlagForge/
├── README.md
├── docker-compose.yml            # postgres:16, 端口 55432:5432（避开本机已装 PG）
├── Makefile                      # dev / test / seed / migrate-up / migrate-down / smoke
├── .gitignore                    # node_modules / .env / bin 等
├── backend/
│   ├── go.mod
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/config.go      # DATABASE_URL 等环境变量
│   │   ├── model/                # GORM models（与迁移 SQL 严格一致）
│   │   ├── repo/                 # 数据访问；PG 错误→业务错误码的唯一映射处
│   │   ├── service/              # 业务逻辑；★ 唯一事务边界
│   │   ├── eval/                 # ★ 评估引擎：纯函数，禁止 import GORM/Gin
│   │   ├── api/                  # Gin router/handlers/中间件/错误信封
│   │   └── dto/                  # 请求/响应结构体 + 校验
│   ├── migrations/               # 0001_init.up.sql / 0001_init.down.sql
│   └── seed/main.go              # 演示数据（幂等）
├── frontend/
│   └── src/
│       ├── api/client.ts         # 按契约封装的 fetch 层 + 错误信封解析
│       ├── mocks/handlers.ts     # MSW handlers（按契约）
│       ├── pages/
│       │   ├── FlagListPage/
│       │   ├── FlagEditPage/     # 新建与编辑共用
│       │   ├── FlagDetailPage/   # 信息 + 规则 Tab + 历史 Tab
│       │   └── ConsolePage/      # 单次评估 Tab + 批量模拟 Tab
│       └── components/           # ValueDisplay / TraceTable / DiffView 等
├── scripts/smoke.sh              # curl 冒烟序列（T5 用）
├── docs/plan/                    # 本计划包
└── docs/ai-logs/                 # AI 日志三层归档（见 00-README）
```

## 4. 数据模型

### 4.1 DDL（迁移文件唯一事实源）

```sql
-- 0001_init.up.sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid（PG16 内置，保险起见）

CREATE TABLE flags (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 100),
    key           TEXT NOT NULL CHECK (key ~ '^[a-z][a-z0-9._-]{1,63}$'),
    environment   TEXT NOT NULL CHECK (environment IN ('development','staging','production')),
    enabled       BOOLEAN NOT NULL DEFAULT true,
    default_value BOOLEAN NOT NULL DEFAULT false,
    description   TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_flags_env_key UNIQUE (environment, key)      -- ★ 题目硬性要求
);

CREATE TABLE rules (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id        UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    attribute      TEXT NOT NULL CHECK (length(trim(attribute)) BETWEEN 1 AND 64),
    operator       TEXT NOT NULL CHECK (operator IN ('equals','in')),
    expected_value JSONB NOT NULL,
    roll_value     BOOLEAN NOT NULL,
    priority       INT NOT NULL CHECK (priority >= 1),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_rules_flag_priority UNIQUE (flag_id, priority)  -- ★ 重复优先级=拒绝
);

CREATE TABLE histories (
    id         BIGSERIAL PRIMARY KEY,
    flag_id    UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    action     TEXT NOT NULL CHECK (action IN (
        'flag_created','flag_updated','flag_enabled','flag_disabled',
        'rule_created','rule_updated','rule_deleted','flag_copied')),
    operator   TEXT NOT NULL DEFAULT 'admin',
    summary    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_histories_flag ON histories (flag_id, created_at DESC);
```

### 4.2 GORM 模型要点

- `Flag`/`Rule`/`History` 三个 struct，字段与 DDL 一一对应；时间用 `time.Time`。
- `expected_value`/`summary` 在 Go 侧用 `datatypes.JSON`（gorm.io/datatypes）或 `json.RawMessage`，
  **禁止**反序列化成 `map[string]any` 存库。
- `updated_at` 由 GORM 自动维护（`autoUpdateTime` tag）。

## 5. 十个设计决策（README 与答辩的核心素材）

| # | 决策 | 理由 |
|---|------|------|
| D1 | 优先级：数字越小优先级越高（1 = 最高），按 priority 升序依次评估 | 业界惯例，UI 与 README 显式标注 |
| D2 | **重复优先级：拒绝**。DB 层 `UNIQUE(flag_id, priority)`，业务层返回 409 | 「第一条命中」语义不含混；策略可测可文档化 |
| D3 | **JSON 类型严格匹配**：`"1"` ≠ `1`，`true` ≠ `"true"`，`null` 不允许作为期望值 | 宽松类型转换是隐藏 bug 温泉；规则可预测 |
| D4 | **停用 = 短路**：`enabled=false` → 恒返回 false，规则完全不参与评估 | 题目原文「Flag 被停用时始终返回 false」的最强解释 |
| D5 | 用户属性只支持**顶层扁平 key**；属性缺失 → 该规则跳过（trace 记录），非错误 | 缺失是正常业务态；嵌套不支持写入 README |
| D6 | flag 的 `key` 与 `environment` **创建后不可变**；编辑仅限 name/description/default_value | key 是调用方锚点，「改 key = 建新 flag」 |
| D7 | **不实现 flag 删除**（题目未要求） | 有意裁剪，README 声明，展示 scoping 判断 |
| D8 | 所有业务变更 + 历史写入在**同一 DB 事务**；service 层是唯一事务边界 | 题目原子性硬要求 |
| D9 | 不做乐观锁/版本号（并发编辑为 last-write-wins），README 写为已知限制 | 可选项不计分；风险隔离 |
| D10 | 全链路 UTC（TIMESTAMPTZ），前端本地化展示 | 时区口径唯一 |

## 6. API 契约（v1）

通用约定：
- Base path `/api/v1`；JSON only；`GET /healthz` 返回 `{"status":"ok"}`。
- **统一错误信封**（所有非 2XX）：

```json
{ "error": { "code": "DUPLICATE_FLAG_KEY", "message": "key 'x' already exists in production", "details": {} } }
```

- 成功响应直接返回数据本体（不包 `{data:...}` 信封）；列表分页响应：
  `{ "items": [...], "page": 1, "page_size": 20, "total": 42 }`

### 6.1 Flag

```
POST /api/v1/flags                     创建
  req : { name*, key*, environment*, default_value*, description, enabled? }
  201  : 完整 flag 对象（含 id, created_at, updated_at）
  400  : VALIDATION_ERROR / INVALID_KEY / INVALID_ENVIRONMENT
  409  : DUPLICATE_FLAG_KEY
  副作用: histories + flag_created

GET /api/v1/flags?environment=&enabled=&q=&page=1&page_size=20
  q 匹配 name 或 key（ILIKE %q%）；page_size 上限 100，默认 20
  200  : { items, page, page_size, total }
  每项附带 rule_count（COUNT 子查询）

GET /api/v1/flags/:id
  200  : { flag 对象, rules: [按 priority 升序] }
  404  : FLAG_NOT_FOUND

PATCH /api/v1/flags/:id                编辑基本信息
  req : { name?, description?, default_value? }   // 传 key/environment → 400 IMMUTABLE_FIELD
  200  : 更新后对象
  副作用: histories + flag_updated（仅记录实际变化的字段）

PATCH /api/v1/flags/:id/status
  req : { enabled* }
  200  : 更新后对象
  副作用: histories + flag_enabled / flag_disabled

GET /api/v1/flags/:id/history?page=1&page_size=20
  200  : { items: [{id, action, operator, summary, created_at}], ... }  最新优先
```

### 6.2 规则

```
POST /api/v1/flags/:id/rules
  req : { attribute*, operator*('equals'|'in'), expected_value*, roll_value*, priority*(≥1) }
  expected_value 校验：
    equals → 必须是标量（string|number|bool，禁止 null/对象/数组）
    in     → 必须是非空数组，元素必须是标量
  201  : 规则对象
  400  : VALIDATION_ERROR（含 expected_value 类型不符）
  404  : FLAG_NOT_FOUND
  409  : DUPLICATE_PRIORITY
  副作用: histories + rule_created

PATCH /api/v1/flags/:id/rules/:rid     同上校验；409 同上；histories + rule_updated（记变化字段）
DELETE /api/v1/flags/:id/rules/:rid    204；histories + rule_deleted（记被删规则摘要）
```

### 6.3 评估（核心）

```
POST /api/v1/evaluate
  req : { flag_key*, environment*, user_attributes* }   // user_attributes 必须是 JSON object
  200 :
  {
    "flag_key": "checkout-redesign-2026",
    "environment": "production",
    "enabled": true,
    "value": true,
    "reason": { "type": "rule_hit",                       // rule_hit | default | disabled
                "rule_id": "…", "priority": 2,
                "summary": "plan equals \"pro\" → true" },
    "trace": [
      { "rule_id": "…", "priority": 1, "attribute": "country", "operator": "in",
        "matched": false, "cause": "value_not_in_list" },
      { "rule_id": "…", "priority": 2, "attribute": "plan", "operator": "equals",
        "matched": true,  "cause": "matched" }
    ],
    "evaluated_at": "2026-10-09T12:00:00Z"
  }
  400 : INVALID_JSON（body 语法错，message 带位置）/ INVALID_USER_ATTRIBUTES / VALIDATION_ERROR
  404 : FLAG_NOT_FOUND（按 key+env 查不到）
```

**Trace 语义**：
- 停用短路 → `value:false, reason.type:"disabled"`，trace 只有一条
  `{ "cause": "short_circuited_by_disabled_flag", "matched": false }`（无 rule_id）。
- 命中即停：trace 只含「已评估」的规则，命中规则之后的规则**不出现**。
- 无规则 / 全部跳过 → `reason.type:"default"`，trace 含每条被跳过规则及原因。
- cause 枚举：`matched | attribute_missing | value_not_equal | value_not_in_list | type_mismatch | short_circuited_by_disabled_flag`
  （`type_mismatch` 仅 equals 产生；in 一律报 `value_not_in_list`，避免组合爆炸）。

### 6.4 批量模拟（亮点 B）

```
POST /api/v1/evaluate/batch
  req : { flag_key*, environment*, user_attributes_list*: [attrs, attrs, …] }   // 1~100 组
  200 :
  {
    "results": [ { "index": 0, "value": true,
                   "reason": {…同上…}, "trace": [...] } ],
    "summary": {
      "total": 5, "true_count": 2, "false_count": 3,
      "default_count": 1,
      "hit_rules": [ { "rule_id": "…", "priority": 2, "attribute": "plan", "count": 2 } ]
    }
  }
  400 : BATCH_LIMIT_EXCEEDED（>100）/ 其余同 evaluate；任一组非法 → 整体 400，details 标注 index
```

### 6.5 环境间配置复制（亮点 D，stretch）

```
POST /api/v1/flags/:id/copy
  req : { target_environment* }
  规则: target != source（否则 400 COPY_TARGET_CONFLICT）；目标环境已有同 key → 409 DUPLICATE_FLAG_KEY
  201 : 新 flag 对象（★ enabled=false 安全默认），规则全量复制（priority 保持）
  副作用: histories + flag_copied（summary 记 source flag_id/environment）
```

### 6.6 错误码表（全量，repo 层唯一映射处）

| code | HTTP | 触发 |
|------|------|------|
| VALIDATION_ERROR | 400 | 字段校验失败，details: `[{field, reason}]` |
| INVALID_KEY | 400 | key 不符 `^[a-z][a-z0-9._-]{1,63}$` |
| INVALID_ENVIRONMENT | 400 | environment 非三枚举 |
| IMMUTABLE_FIELD | 400 | PATCH 尝试改 key / environment |
| INVALID_JSON | 400 | body JSON 语法错误（message 含位置） |
| INVALID_USER_ATTRIBUTES | 400 | user_attributes 非 JSON object |
| BATCH_LIMIT_EXCEEDED | 400 | 批量组数 > 100 |
| COPY_TARGET_CONFLICT | 400 | 复制目标环境 = 源环境 |
| FLAG_NOT_FOUND | 404 | flag / flag_key+env 不存在 |
| RULE_NOT_FOUND | 404 | 规则不存在或不属于该 flag |
| DUPLICATE_FLAG_KEY | 409 | env+key 撞号（PG 23505 / uq_flags_env_key） |
| DUPLICATE_PRIORITY | 409 | priority 撞号（PG 23505 / uq_rules_flag_priority） |
| INTERNAL_ERROR | 500 | 兜底（log 详情，响应不含内部细节） |

## 7. 评估引擎规格（internal/eval）

```go
package eval  // 纯函数：禁止 import GORM/Gin/PG 驱动

type Rule struct {
    ID       string
    Priority int
    Attribute string
    Operator string          // "equals" | "in"
    Expected any             // 已 normalize：equals→标量；in→[]any 标量元素
    RollValue bool
}

type Input struct {
    Enabled      bool
    DefaultValue bool
    Rules        []Rule     // 任意顺序传入，引擎内部按 priority 升序排
    UserAttrs    map[string]any
}

type TraceStep struct {
    RuleID    string  // 停用短路时为空
    Priority  int
    Attribute string
    Operator  string
    Matched   bool
    Cause     string
}

type Result struct {
    Value      bool
    ReasonType string  // "rule_hit" | "default" | "disabled"
    HitRule    *Rule
    Trace      []TraceStep
}

func Evaluate(in Input) Result
func Normalize(v any) any   // 导出供 service 层复用：两侧（DB jsonb / 请求 JSON）统一归一
```

**Normalize 规则（D3 的实现核心）**：
- 两侧值必须经**同一个** `Normalize` 后比较。
- json → `any`：数字一律 `float64`（encoding/json 行为）；`1` 与 `1.0` 归一后相等 —— 一致性达成。
- 类型不同（string vs number vs bool）→ 不相等；equals 场景报 `type_mismatch`。
- 比较用 `reflect.DeepEqual`（两侧已归一，安全）。

**必须覆盖的测试矩阵（T2 验收标准）**：

| # | 用例 | 期望 |
|---|------|------|
| 1 | disabled 短路（即使规则会命中） | false / disabled / trace 仅 1 条短路记录 |
| 2 | 无规则，enabled=true | default_value / default / trace 空 |
| 3 | 规则全 miss | default_value / default / trace 含各跳过原因 |
| 4 | 首条命中即停 | 后续规则不出现在 trace |
| 5 | priority 升序正确（乱序传入） | 按 1,2,3 顺序评估 |
| 6 | equals 命中（string/number/bool 各一） | value=roll_value，cause=matched |
| 7 | equals 不命中 | cause=value_not_equal |
| 8 | equals 类型不匹配（"1" vs 1；true vs "true"） | cause=type_mismatch |
| 9 | in 命中 / 不命中 | matched / value_not_in_list |
| 10 | in 列表混合类型（["pro", 42]）| 仅同类型元素可命中 |
| 11 | 属性缺失 | 该规则 skip，cause=attribute_missing，继续后续 |
| 12 | in 单元素列表 | 正常工作 |
| 13 | default_value=true 的 flag 全 miss | 返回 true / reason=default |
| 14 | 数字归一：attrs `42` vs expected `42` | 命中（双侧 float64） |
| 15 | attrs 含嵌套对象，规则属性指向其顶层 key | 该规则正常比对（对象值永远不等于标量 → skip） |

## 8. 亮点规格

### A. 评估轨迹 Trace
见 §6.3 —— 引擎原生产出，前端 `TraceTable` 组件渲染：每行 priority/attribute/operator/matched/cause，
matched=true 高亮，cause 有中文文案映射表（frontend 常量表）。

### B. 批量模拟分布（ConsolePage 第二个 Tab）
- 输入：textarea，每行一个 JSON object（或直接传 JSON 数组），≤100 行
- 输出：结果表格（index/关键属性摘要/value/reason）+ 统计卡（true/false 数、默认值比例）+
  「命中规则分布」条形列表（按命中次数排序）
- 场景文案（页面引导语）：「上线前模拟：金 V 用户 / CN 用户 / 新用户分别拿到什么？」

### C. 历史变更 Diff（FlagDetailPage 历史 Tab）
history summary 结构化 schema：

```json
flag_created  : { "flag": {name, key, environment, default_value, enabled} }
flag_updated  : { "changes": { "name": {"old":..,"new":..}, "default_value": {"old":false,"new":true} } }
flag_enabled  : { "changes": { "enabled": {"old":false,"new":true} } }
flag_disabled : { "changes": { "enabled": {"old":true,"new":false} } }
rule_created  : { "rule": {priority, attribute, operator, expected_value, roll_value} }
rule_updated  : { "rule_id":.., "priority":.., "changes": {…} }
rule_deleted  : { "rule": {…} }
flag_copied   : { "flag": {…}, "source": {"flag_id":.., "environment":..} }
```

前端 `DiffView` 组件：逐字段 `old → new`（删除线红 / 新值绿）；rule 类操作渲染规则摘要行
（`#2 plan equals "pro" → true`）。

### D. 环境间配置复制（stretch）
见 §6.5。前端：FlagDetailPage 下拉选目标环境 + 确认弹窗（明示「复制后 enabled=false」）。

## 9. 前端页面清单与质量要求

| 页面 | 路由 | 要点 |
|------|------|------|
| Flag 列表 | `/` | env/enabled 筛选 + 搜索（name/key）+ 服务端分页；行内 Switch 启停（乐观更新+失败回滚）；rule_count 列；空态引导创建 |
| 新建/编辑 | `/flags/new`, `/flags/:id/edit` | key 正则即时校验（与后端一致）；编辑态 key/env 只读并说明原因；409 内联到对应字段 |
| 详情 | `/flags/:id` | 基本信息 + 「规则」「历史」Tab；规则表格按 priority 升序；新建/编辑规则 Modal；删除 Popconfirm；优先级 409 内联 |
| 评估控制台 | `/console` | 单次评估 Tab + 批量模拟 Tab；JSON 编辑器（校验+格式化+错误提示）；结果卡（value 大字 + reason 徽标 + TraceTable）；「复制 curl」按钮 |

**每页必备**（题目明说）：加载态、空态、成功/失败 message 反馈、表单校验。
样式不追求精美，追求信息层级清晰。

## 10. 测试策略

| 层 | 范围 | 工具 |
|----|------|------|
| 单测（必做） | `internal/eval` 全部 15 用例 | `go test ./internal/eval/ -v`，秒级，无 DB |
| API 集成（必做） | 关键端点：创建 409 / 优先级 409 / evaluate 400·404·短路·命中·默认 / PATCH 不可变字段 400 / 原子性（规则写入失败→history 无记录） | 连 compose 里的 PG，`go test ./internal/api/ -tags=integration`（或独立 `integration` 包） |
| 前端测试 | 不做（题目可选项，README 声明） | — |

## 11. Seed 演示数据（讲一个故事）

评委打开即有代入感的场景：**「checkout-redesign-2026 结账页重构灰度」**

| flag | env | enabled | default | 规则 |
|------|-----|---------|---------|------|
| checkout-redesign-2026 | production | true | false | #1 country in [US,CA,GB]→true；#2 plan equals "pro"→true；#3 user_id in [9001,9002,9003]→true（内部白名单） |
| checkout-redesign-2026 | development | true | true | #1 country equals "CN"→false（国内环境强制旧行为，演示跨环境同 key 合法） |
| new-search-rank | staging | true | false | #1 plan in ["pro","team"]→true；#2 beta_tester equals true→true |
| billing-portal | development | **false** | true | 一条规则（演示停用短路：规则存在但恒 false） |

Seed 幂等：以 (environment, key) 为准 upsert，history 只在首次创建时写入。

## 12. README 大纲（T5 按此撰写）

1. **简介**：一句话 + 功能清单（必做 ✓ 表 + 四个亮点）
2. **快速开始**：前置依赖（Go/Node/Docker 或本地 PG）→ `make dev` 一条命令（内部：compose up → migrate → seed → 起前后端）；分步骤备选
3. **数据模型**：三表说明 + 关键约束（两个 UNIQUE 的语义）
4. **规则评估语义**（评分重读区）：优先级方向 D1、首条命中、类型严格匹配 D3、停用短路 D4、确定性策略 D2、属性缺失 D5 —— 附 3 个评估示例
5. **API 摘要**：端点表 + curl 示例（创建/评估）
6. **测试**：怎么跑、覆盖说明
7. **设计取舍**：十决策精选表述 + 「不做什么」（删除 D7、乐观锁 D9、嵌套属性）及原因
8. **已知限制**
9. **AI 使用说明**：三层日志结构 + 工作流图（主线架构 agent → 提示词 → 并行 executor → 归档）
