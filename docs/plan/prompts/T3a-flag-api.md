# 任务 T3a：Flag CRUD / 状态 / 历史 API

## 你的角色

你是 FlagForge 的后端业务工程师，负责 flag 生命周期 API + **全项目共用的错误处理与事务基建**
（T3b 会直接复用你立的模式）。事务原子性（业务变更+历史写入）是题目硬性要求，验收会构造失败场景验证。

## 必读（动手前读完）

1. `docs/plan/01-design.md` §5 决策 D6/D8/D9、§6 前言+§6.1+§6.6（API 契约与错误码表）、§8-C（history summary schema）
2. `docs/plan/02-edge-cases.md` 组 1、组 2、组 7、3.13（原子性）
3. T1 报告（repo 层现成能力 + 错误映射）、T0 报告（目录现状）

## 交付物

### 1. 基建 `backend/internal/api/`

- `errors.go`：统一错误信封 + 业务错误类型（code/message/details），错误码枚举与 §6.6 逐字一致
- `respond.go`：`respondError(c, httpStatus, code, msg, details)` / `respondOK` / 分页响应助手
- `router.go`：挂 `/api/v1` 分组 + 各 handler；`GET /healthz` 保留
- handler 只做：参数绑定 → 调 service → 映射响应。**业务逻辑零残留**

### 2. DTO + 校验 `backend/internal/dto/`

- 创建/编辑/状态/列表查询的请求结构体
- 校验规则（组 1）：key 正则 `^[a-z][a-z0-9._-]{1,63}$`（错误码 INVALID_KEY）、
  environment 三枚举（INVALID_ENVIRONMENT）、name trim 后 1~100
- **PATCH 语义陷阱**：判断「key/environment 字段是否出现在 body」≠「值是否为零值」。
  先 `json.Unmarshal` 到 `map[string]json.RawMessage` 检测字段存在性（存在禁改字段 → 400 IMMUTABLE_FIELD），
  再绑定到 DTO 校验可改字段。struct 绑定无法区分「没传」和「传了零值」，这是本任务最大陷阱
- 校验失败统一 `VALIDATION_ERROR` + `details:[{field,reason}]`

### 3. Service `backend/internal/service/flag.go`

- **唯一事务边界**：所有「业务变更 + history 写入」必须包在 `db.Transaction(...)` 内
- history 写入助手：`writeHistory(tx, flagID, action, summary)` —— summary 结构严格按 §8-C：
  - `flag_created`  /  `flag_enabled` / `flag_disabled`（changes 形态）
  - `flag_updated`：**diff 后只记实际变化字段**；无任何变化 → 不写 history（组 1.9、2.2）
- 启停：目标值与现值相同 → 200 但不写 history（组 2.2）
- 复用 repo 的 23505 映射：`DUPLICATE_FLAG_KEY` → 409 + 友好 message
  （如 `key 'checkout-redesign-2026' already exists in production`）

### 4. 集成测试 `backend/integration/`（或 `internal/api` + build tag）

连 compose 里的真实 PG（每用例用独立随机数据，测试间不共享状态），至少覆盖：

- 创建成功 → 201 + history 有 flag_created
- 同 env 撞 key → 409（组 1.1）；跨 env 同 key → 201（组 1.2）
- PATCH 传 key → 400 IMMUTABLE_FIELD（组 1.7）
- PATCH 改 name → history.changes 仅含 name（组 1.8）
- 停用两次 → 第二次无新 history（组 2.2）
- 列表筛选三条件组合 + 分页边界（组 2.3/2.5）
- **原子性（3.13）**：可用手法——在测试里注入一个使 history 写入失败的路径
  （如临时超长 summary 触发约束/ mock repo 返回错误），断言业务变更也未落库

## 验收标准

- [ ] 上述集成测试全绿：`go test ./... -tags=integration`（连 compose PG）
- [ ] curl 冒烟：创建→列表→详情→改 name→停用→history，响应与 §6.1 契约逐字段一致
- [ ] 所有错误响应走统一信封，无裸 `{"error":"..."}` 字符串
- [ ] handler 文件中无 SQL/事务代码（grep 自查 `\.Begin\|\.Transaction` 只出现在 service/repo）

## 硬约束

- 不实现规则 CRUD / evaluate / copy（T3b 范围），router 里不留死路由
- D9：不加乐观锁，并发编辑 last-write-wins —— 不做额外机制
- 发现 T1 repo 层缺方法：可以加，但保持「错误映射只在 repo 层」原则，改动写进报告

## 完成后必做

写 `docs/ai-logs/T3a-report.md`（模板同 T0；「关键决策」必须包含：PATCH 字段存在性检测的实现方式、
原子性测试的注入手法），然后 `git commit -m "feat(T3a): flag CRUD, status and history APIs with transactional audit"`。
