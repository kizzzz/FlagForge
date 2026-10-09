归档日期：2026-10-09（Asia/Shanghai）
原文来源：docs/plan/prompts/T1-db-models.md

# 任务 T1：数据库迁移与模型层

## 你的角色

你是 FlagForge 的数据层工程师：把设计定稿的 schema 落成迁移文件，建 GORM 模型与 repo 基础层，
并写幂等的演示数据。schema 质量是本项目评分基本盘之一（两个 UNIQUE 约束是题目硬性要求）。

## 必读（动手前读完）

1. `docs/plan/01-design.md` §4 数据模型（DDL 原样落地，**不改一字**）、§5 设计决策 D1~D3、§11 seed
2. `docs/plan/02-edge-cases.md` 组 1（1.1/1.2 涉及 DB 约束）、组 10（迁移可逆/seed 幂等）
3. T0 执行报告 `docs/ai-logs/T0-report.md`（了解环境与目录现状）

## 交付物

### 1. 迁移文件 `backend/migrations/`

- `0001_init.up.sql`：**逐字复制** 01-design §4.1 的 DDL
- `0001_init.down.sql`：按依赖倒序 DROP（histories → rules → flags），可完整回退
- 命名符合 golang-migrate 规范；Makefile/scripts 里的 migrate 命令保持可用

### 2. GORM 模型 `backend/internal/model/`

- `Flag` / `Rule` / `History` 三个 struct，字段与 DDL 一一对应（列名用 tag 显式指定，不信 snake_case 推断）
- `expected_value`、`summary` 用 `datatypes.JSON`
- `updated_at` 用 `autoUpdateTime`；`created_at` 用 `autoCreateTime`
- **全局禁用 AutoMigrate**（main.go 里不得出现）；schema 唯一归属迁移文件

### 3. Repo 基础层 `backend/internal/repo/`

- `FlagRepo` / `RuleRepo` / `HistoryRepo`：基础 CRUD + 以下定制方法
  - `FlagRepo.List(environment, enabled *string/q string, page, pageSize int)` —— q 对 name/key ILIKE，
    返回 items + total，items 带 `rule_count`（COUNT 子查询）
  - `FlagRepo.GetByKeyEnv(key, environment)` —— evaluate 用
  - `RuleRepo.ListByFlagOrdered(flagID)` —— priority 升序
  - `HistoryRepo.Create` / `ListByFlag(flagID, page)` —— created_at DESC
- **PG 错误映射**（本层是唯一映射处，T3a/T3b 复用）：

```go
// repo/errors.go —— 23505 唯一违反 → 按约束名映射业务错误
// uq_flags_env_key       → DUPLICATE_FLAG_KEY
// uq_rules_flag_priority → DUPLICATE_PRIORITY
// 其他 23505              → INTERNAL_ERROR
// 用 errors.As(*pgconn.PgError) 判断，不 parse 错误字符串
```

### 4. Seed `backend/seed/main.go`

- 数据按 01-design §11 表格**逐行落地**（4 个 flag：production/development 的
  checkout-redesign-2026、staging 的 new-search-rank、development 的 billing-portal）
- 幂等：以 (environment, key) `ON CONFLICT DO NOTHING`；**仅首次创建**时写 flag_created history
  （重复执行 seed 不产生新 history、不改已有数据）
- 规则的 expected_value 用具体 JSON（如 `["US","CA","GB"]`、`"pro"`、`[9001,9002,9003]`）

## 本任务已知陷阱

- 迁移 up/down 必须可逆：`migrate-down` 后再 `migrate-up` 无残留对象（自验组 10.1）
- 约束名是错误映射的契约，拼写与设计文档逐字符一致
- seed 幂等 ≠ 跳过已有：已有 flag 的规则缺失时要不要补？——**不补**，seed 只管首次创建（简化口径，写进报告）
- Windows Git Bash 下 golang-migrate CLI 安装：`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`，
  装不上就报告，Makefile 提供基于库 API 的 `cmd/migrate/main.go` 备选

## 验收标准

- [ ] `migrate-up` / `migrate-down` / 再 `migrate-up` 全部成功（组 10.1）
- [ ] psql 或 Go 查询确认三表结构、两个 UNIQUE 约束、CHECK 约束存在且名字正确
- [ ] seed 跑两次：第二次无新增 flag、无新增 history（组 10.2）
- [ ] 手工 INSERT 撞 `uq_flags_env_key` 与 `uq_rules_flag_priority`，确认 PG 报 23505 且约束名正确
- [ ] `go build ./...` 与 `go vet ./...` 通过

## 硬约束

- DDL 与设计文档逐字一致；发现 DDL 疑似 bug → 停止并写入报告「需人工决策」，不擅自修改
- repo 层不做业务判断（事务、history 语义都在 T3a/T3b 的 service 层）

## 完成后必做

写 `docs/ai-logs/T1-report.md`（模板同 T0：做了什么/关键决策/问题/未尽事项/验收打勾+输出摘要），
然后 `git commit -m "feat(T1): database migrations, models, repo layer and seed"`。
