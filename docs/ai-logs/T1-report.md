# T1 执行报告

执行日期：2026-10-09（Asia/Shanghai）

项目 worktree：`C:/Users/77230/FlagForge/.worktrees/t1`

分支：`codex/t1-db-models`；基线：`9a3bf66`

状态：**T1 实现完成，五项任务验收全部通过。**

## 做了什么

- 阅读 T1 原始提示词、01-design §4/§5 D1~D3/§11、02-edge-cases 1.1/1.2/组10、T0-report；完整提示词与归档日期保存于 `T1-prompt.md`。
- `backend/migrations/0001_init.up.sql` 逐字提取 01-design §4.1 SQL 代码块，包含原注释、空行和约束名；down 依次 DROP histories、rules、flags。
- `backend/internal/model/models.go`：三表 24 个字段全部对应；UUID 使用 string，BIGSERIAL 使用 int64，JSONB 使用 datatypes.JSON，列名显式声明，时间自动创建/更新。没有调用 AutoMigrate。
- `backend/internal/repo/`：Flag CRUD、env/enabled/q 组合筛选和分页总数、COUNT 子查询 rule_count、key+env 查询；Rule CRUD 按 flag_id 限定，按 priority 升序查询；History 创建与最新优先分页。
- `repo/errors.go` 使用 errors.As 检查 pgx/v5/pgconn.PgError，按 23505 的 ConstraintName 映射三个业务码；不解析错误字符串；保留底层原因供日志和 errors.Is/As 使用。
- `backend/seed/main.go`：四个 flag、七条具体 JSON 规则、首次 flag_created 历史；四个演示项在一个初始化事务内写入。DATABASE_URL 与 T0 默认值一致，seed 时间使用 UTC。
- 依赖仅补齐 pgx v5.5.5 与现有 GORM/datatypes 所需间接依赖和校验和；保留全部 T0 固定依赖及 `go 1.22`，没有运行 tidy 删除其他任务依赖。
- 先写错误映射、repo 和 seed 测试，补齐依赖后运行确认缺少目标 API 的失败，再实现并运行通过；补充真实数据库唯一冲突与启用 true→false 的回归覆盖。

## 关键决策

1. bool 字段不带 GORM `default:true`，让显式 false 参与 INSERT；更新接收 map[string]any，让 false 和空字符串参与 UPDATE。集成测试验证 flag 初始 false、enabled true→false、default true→false、description 非空→空以及 rule.roll_value true→false。
2. repo 不开启业务事务、不自动写历史、不承担字段合法性/不可变字段/分页参数校验。后续 service 在 db.Transaction 内把 tx 传入 repo 构造器，负责允许字段、分页口径和业务历史。
3. seed 对 `(environment,key)` 使用 ON CONFLICT DO NOTHING；RowsAffected=0 时完全跳过。已有 flag 的缺失规则不补，已有字段不覆盖，历史不增加。
4. §11 对 billing-portal 仅指定「一条规则」；落为 priority=1、plan equals `"pro"`→true，使停用短路演示具备会命中的规则。
5. 列表返回初始化空 slice，分页超出范围时序列化为 []。Flag 列表按 created_at DESC / id ASC，历史按 created_at DESC / id DESC，时间相同时仍稳定；规则依赖 UNIQUE 保证 priority 唯一。
6. 非23505错误原样返回，包含 gorm.ErrRecordNotFound；查询或更新/删除不存在记录时由 service 映射相应 404。其他23505映射 INTERNAL_ERROR。
7. down 按任务要求只 DROP 三张业务表；pgcrypto 是共享基础扩展，予以保留，schema_migrations 由迁移工具管理并保留。业务表及其依赖索引/序列可完整回退，不声称撤销全部数据库对象。

## 给 T3a / T3b 的公共接口

所有方法的第一个参数均为 `ctx context.Context`；构造器接收 `db *gorm.DB`，可以是事务句柄。更新 map 的 key 使用数据库列名。

```go
NewFlagRepo(db) *FlagRepo
FlagRepo.Create(ctx, *model.Flag) error
FlagRepo.GetByID(ctx, id string) (*model.Flag, error)
FlagRepo.GetByKeyEnv(ctx, key, environment string) (*model.Flag, error)
FlagRepo.Update(ctx, id string, fields map[string]any) error
FlagRepo.Delete(ctx, id string) error
FlagRepo.List(ctx, environment string, enabled *string, q string, page, pageSize int) ([]FlagListItem, int64, error)
// FlagListItem 嵌入 model.Flag，增加 RuleCount int64 / json:"rule_count"

NewRuleRepo(db) *RuleRepo
RuleRepo.Create(ctx, *model.Rule) error
RuleRepo.GetByID(ctx, flagID, id string) (*model.Rule, error)
RuleRepo.ListByFlagOrdered(ctx, flagID string) ([]model.Rule, error)
RuleRepo.Update(ctx, flagID, id string, fields map[string]any) error
RuleRepo.Delete(ctx, flagID, id string) error

NewHistoryRepo(db) *HistoryRepo
HistoryRepo.Create(ctx, *model.History) error
HistoryRepo.ListByFlag(ctx, flagID string, page, pageSize int) ([]model.History, int64, error)

MapError(err error) error
// errors.As(err, &repoError) 后读取 (*repo.Error).Code；Unwrap 返回底层 Cause
// repo.DuplicateFlagKey / repo.DuplicatePriority / repo.InternalError
```

`enabled=nil` 表示不筛选；非nil字符串应由 service/API 先验证为 "true" 或 "false"。page/pageSize 的默认值、上限和合法性同样由 service/API 处理。FlagRepo.Delete 是基础数据操作；产品不新增 Flag 删除 API（D7）。History 作为审计记录仅提供 Create/List，不提供业务编辑删除入口。服务端建 GORM 连接时应配置 NowFunc 返回 UTC，与 seed 及 D10 一致。

## 遇到的问题与解决

- 首次测试缺少 pgx、GORM/datatypes 的 go.sum 项；使用 GOPROXY=https://goproxy.cn,direct 和 T0 固定版本补齐依赖后继续。没有升级 Go 指令或其他依赖。
- 最初测试因待实现 MapError、model/repo API、runSeed 不存在而失败；实现后单测与真实 PG 集成测试均通过。
- Git 提示 Windows checkout 的 LF/CRLF 转换；迁移内容以设计 SQL 提取值做逐字符比较验证通过，git diff --check 没有空白错误。
- 手工唯一冲突命令的 exit=1 是预期结果；psql verbose 输出明确确认23505和对应约束名。两次冲突均未插入数据。
- 最终恢复时曾从仓库根执行 go run ./backend/seed，因根目录无 go.mod 失败；立即改为 backend workdir 下 go run ./seed 成功，数据库恢复演示数据。

## 未尽事项 / 需要人工决策

- **T1 无未完成项，无待人工决策，未发现需要停止上报的 DDL 问题。**
- 业务 API、service 事务与历史语义、eval、前端不在本任务范围。合并前代码审查交由 root，遵守本 executor 禁止创建其他 agent 的任务限制。
- 本次没有删除 Docker 数据卷或重建容器；验证了现有 PG 的迁移 up/down/up。全新环境 `docker compose down -v && up`、全应用 dev/smoke 等属于 T5 最终复现验收，未将其标为通过。
- PostgreSQL 留在迁移版本1，演示数据为4 flags / 7 rules / 4 histories。集成测试全部使用外层事务回滚，临时 CHECK 和删除数据也随事务回滚，无残留测试规则或约束。

## 验收自验清单与输出摘要

- [x] migrate up / down / up：在本 worktree 调用 Git Bash scripts/migrate-up.sh、migrate-down.sh、再 migrate-up.sh；COMPOSE_PROJECT_NAME=flagforge。分别输出 `1/u init`、`1/d init`、`1/u init`，全部 exit=0。down 后 pg_tables 只剩 schema_migrations。补充第二次 down 查询 pg_class，三张业务表、三个PK索引、两个UNIQUE索引、history索引和 histories_id_seq 共10个业务关系对象剩余数为0，pgcrypto 扩展保留数为1；随后 up 和 seed 恢复成功。
- [x] 数据库表结构/约束：information_schema.columns 返回24列，类型/非空/默认值与DDL对应；pg_constraint 返回14个约束，包括 `uq_flags_env_key`、`uq_rules_flag_priority`、7个CHECK、3个PK、2个CASCADE FK；pg_indexes 包含 `idx_histories_flag (flag_id, created_at DESC)`。SQL 文本与 §4.1 代码块逐字符比较 `DDL exact text match: PASS`。
- [x] seed 两次：`go run ./seed` 均 exit=0、输出 seed complete；每次查询均为 flags=4 / rules=7 / histories=4。billing-portal development 为 enabled=f / default_value=t；查询七条规则的JSON与§11对应。
- [x] 手工 INSERT 撞两个UNIQUE：psql 使用 ON_ERROR_STOP=1、VERBOSITY=verbose；重复production checkout key得到 `ERROR: 23505` 和 `CONSTRAINT NAME: uq_flags_env_key`；对其 priority=1 重复插入规则得到 `ERROR: 23505` 和 `CONSTRAINT NAME: uq_rules_flag_priority`。
- [x] `go build ./...`：exit=0，零输出；`go vet ./...`：exit=0，零输出。
- [x] `DATABASE_URL=... go test -tags=integration ./... -count=1 -v`：exit=0，repo与seed全通过，包括包装PG错误、未知23505兜底、错误字符串不误判、组合筛选/ILIKE/rule_count/空页、零值CRUD、规则归属与排序、历史分页、真实唯一冲突映射、seed幂等不补规则、history CHECK失败后seed整批回滚。未配置DATABASE_URL的普通测试不会接触数据库。

所有 Docker/Compose 查询明确使用 `-p flagforge -f C:/Users/77230/FlagForge/docker-compose.yml`，没有启动绑定55432的第二个容器。

提交消息：`feat(T1): database migrations, models, repo layer and seed`。
