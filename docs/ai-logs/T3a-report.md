# T3a 执行报告

执行日期：2026-10-10（Asia/Shanghai）

worktree：`C:/Users/77230/FlagForge/.worktrees/t3a`；分支：`codex/t3a-flag-api`；基线：`1769032`。

状态：**DONE，T3a 实现与自验完成；由 root 进行规格与质量审查后合并。**

## 做了什么

- 阅读 T3a 全提示词、01-design §5 D6/D8/D9、§6 前言/6.1/6.6、§8-C、02-edge-cases 组1/2/7与3.13，以及 T0/T1 报告。完整原始提示词与本轮补充上下文归档于 T3a-prompt.md。
- `internal/api/` 实现六个生命周期端点：创建、列表、详情、编辑、启停、历史分页。保留 healthz；路由没有规则/evaluate/copy 的占位入口。
- 统一错误信封包含 `error.code/message/details`，§6.6 的全部13个 code 完整定义。已知业务错误按状态表返回；内部错误记录日志但响应仅为 INTERNAL_ERROR/internal server error。未知路由、未知方法和 panic 均保持 JSON 信封。
- `internal/dto/flag.go` 定义请求与响应结构、必填布尔指针、三环境枚举、key 正则、Unicode name trim/字符长度、JSON 对象与字段类型/非空校验、UUID 绑定校验、AND 列表查询与安全分页参数。
- `internal/service/flag.go` 在同一 PG 事务中持久化业务与历史；字段实际 diff、空 diff 不写历史，重复启停不更新也不写历史。创建 enabled 默认 true，description 默认空串，false 与空字符串可正常落库。
- 详情 rules 按 priority 升序；列表 rule_count 使用已有 repo COUNT 子查询；历史只返回契约规定的五个字段，按 created_at DESC/id DESC 分页，空 items 返回 []。
- `cmd/server/main.go` 实际初始化 GORM/PG，Ping 后挂业务 router，UTC NowFunc。默认仍监听 :8787；可选通用 SERVER_ADDR 支持部署/临时自有端口。
- 新增真实 PG API 集成测试和 DTO 单测；没有修改 repo、迁移、seed、eval、frontend、README、根 scripts、go.mod/go.sum，也没有增加依赖。

## 关键决策

1. **PATCH 字段存在性**：先 `json.Unmarshal` 到 `map[string]json.RawMessage`，对 key/environment 做 map lookup；一旦字段存在立即 IMMUTABLE_FIELD，包含相同现值、空串、null、非字符串以及转义字段名。随后才反序列化 UpdateFlag、校验可改字段。Name/Description/DefaultValue 使用指针保留缺省语义；显式 null 拒绝。空对象与全部同值编辑为200、无新历史。
2. **事务唯一边界**：Create/Update/Status 各仅在 service 调用 `db.WithContext(ctx).Transaction(...)`；构造 repo 时传 tx，WriteHistory 使用同一 tx，不开启或提交事务。handler 只负责绑定、调用、响应。没有测试专用 hook 或生产失败注入入口。
3. **PG 原子性注入**：每个集成用例创建 UUID 随机 schema，将原始 SQL 迁移建于该 schema；每个连接设置自己的 search_path。测试给隔离 histories 添加 `CHECK (action = 'flag_created')`，令编辑/启停 history INSERT 真实触发23514，断言 name/enabled/历史总数均回滚。再改为 `CHECK (false) NOT VALID`，令创建后的 history 写入失败，断言新 flag 不存在。最后关闭连接并 DROP 本例 schema。没有改 public 的约束和时区，没有影响其他测试或演示数据。
4. **错误分层**：`internal/apperror` 是 DTO/service/API 共用的业务错误与码定义，避免 api→service→dto→api 循环；api/errors.go 用类型/常量别名重新导出。PG23505/约束名仍仅在 repo.MapError 映射；service 只按已映射的 DuplicateFlagKey 生成友好409消息。
5. **UUID 与分页**：UUID 要求标准带连字符表示，大小写均可；拒绝 uuid.Parse 额外接受、但 PG 不接受的 URN，避免22P02变500。page/page_size 仅接受整数，重复值拒绝；0/缺省使用1与20；page_size>100截100；负数、超int整数、offset计算溢出返回400。无乐观锁或版本字段，维持 D9 的 last-write-wins。
6. **UTC 回读归一**：首次实际 server curl 发现 history 返回 `2026-10-10T00:56:51.914297+08:00`。pgx v5.5.5 二进制 TIMESTAMPTZ codec 使用 `time.Unix` 回读为本地时区，NowFuncUTC只能约束写入。service 输出 flag 的 CreatedAt/UpdatedAt、详情 rule 的两时间及 history.CreatedAt 显式 `.UTC()`；列表、编辑、状态也覆盖。没有改进程全局 time.Local 或数据库全局时区。

## 给 T3b 的公共接口

所有 service/repo 调用通过请求 context；T3b 可以直接在 `RegisterRoutes` 同一个 `/api/v1` 组追加其路由。

```go
// internal/api
NewRouter(db *gorm.DB) *gin.Engine
RegisterRoutes(router *gin.Engine, db *gorm.DB)
NewFlagHandler(s *service.FlagService) *FlagHandler
ReadBody(c *gin.Context) ([]byte, error)
HandleError(c *gin.Context, err error)
RespondError(c *gin.Context, status int, code, message string, details any)
RespondOK(c *gin.Context, status int, value any)
RespondPage(c *gin.Context, items any, page, pageSize int, total int64)
// api.Error/FieldError aliases apperror.Error/FieldError;
// api exports all 13 code constants using CamelCase names.

// internal/apperror
type Error struct { Code string; Message string; Details any; Cause error }
type FieldError struct { Field string; Reason string }
Validation(field, reason string) *Error
HTTPStatus(code string) int

// internal/dto
DecodeObject(body []byte) (map[string]json.RawMessage, error)
DecodeFields(body []byte, fields map[string]json.RawMessage, target any, allowed ...string) error
ValidateKey(key string) error
ValidateEnvironment(environment string) error
ValidateID(id string) error
ParsePageQuery(values url.Values) (PageQuery, error)
ParseListFlags(values url.Values) (ListFlags, error)
DecodeCreateFlag(body []byte) (CreateFlag, error)
DecodeUpdateFlag(body []byte) (UpdateFlag, error)
DecodeFlagStatus(body []byte) (FlagStatus, error)
// CreateFlag/UpdateFlag/FlagStatus also expose Validate() error.
// PageQuery{Page,PageSize}; ListFlags embeds PageQuery and adds Environment/Enabled/Q.

// internal/service
NewFlagService(db *gorm.DB) *FlagService
(*FlagService).Create(ctx context.Context, req dto.CreateFlag) (*model.Flag, error)
(*FlagService).Detail(ctx context.Context, id string) (*dto.FlagDetail, error)
(*FlagService).List(ctx context.Context, req dto.ListFlags) ([]repo.FlagListItem, int64, error)
(*FlagService).Update(ctx context.Context, id string, req dto.UpdateFlag) (*model.Flag, error)
(*FlagService).Status(ctx context.Context, id string, req dto.FlagStatus) (*model.Flag, error)
(*FlagService).History(ctx context.Context, id string, page dto.PageQuery) ([]dto.HistoryItem, int64, error)
WriteHistory(ctx context.Context, tx *gorm.DB, flagID, action string, summary any) error
type Change struct { Old any; New any } // JSON old/new
```

T3b 使用 `apperror.Error{Code:..., Message:..., Details:...}` 返回业务错误；未知 error 交 api.HandleError 记录并隐藏细节。WriteHistory 只序列化 summary、通过 repo 写入，必须由调用方先开启 service 事务；summary 按 §8-C 组装。T3b 规则与复制返回模型的 CreatedAt/UpdatedAt 也须显式 `.UTC()`，与本轮 flagUTC/详情 rules 的做法相同。DTO DecodeFields 拒绝 unknown/null 字段；若规则 expected_value 或评估 user_attributes 需专门类型错误，应先处理 RawMessage，再调用适合的通用校验。

## 红 → 绿证据

- 首次保留仅 healthz 的可运行 router 骨架，先写六组真实 PG 端到端断言。`go test ./internal/api -tags=integration -count=1 -v` exit=1，六组均在请求处失败，典型 `expected:201 actual:404`；不是依赖缺失或编译错误。
- 生命周期与事务实现后同命令 exit=0，六组全 PASS，含 history失败注入后的真实回滚。
- 分页补充测试先红：`go test ./internal/dto -count=1 -v` exit=1，`page=1&page=2` / `page_size=20&page_size=50` 错误地被接受；加入重复检查后同测试 PASS。
- UUID 回归先红：URN 请求导致 PG22P02、响应500；标准表示校验后返回400 VALIDATION_ERROR，完整集成测试 PASS。
- UTC 回归先红：使用默认二进制 PG 驱动模式，`TestCreateAndUniqueEnvironment` 在 history 读取断言 `created_at must use UTC Z: ...+08:00` 失败；service 归一后递归 UTC 断言覆盖所有响应，完整测试 PASS。

## 验收命令与结果

在 worktree 的 backend 目录，以 PowerShell 设置 DATABASE_URL 为已有55432 PG、GIN_MODE=release，Go 使用 `C:/Program Files/Go/bin/go.exe`：

- `go test ./... -tags=integration -count=1`：exit=0。最终输出 api 7.925s、dto 0.274s、eval 0.233s、repo 0.753s、seed 0.561s，均 ok；所有 API 用例使用隔离 schema。
- `go test ./... -count=1`：exit=0，普通 DTO/eval/repo 单测均 ok；不带 integration 不连接数据库。
- `go build ./...`、`go build -o bin/t3a-server.exe ./cmd/server`、`go vet ./...`：exit=0，无诊断。
- `rg -n '\.Begin\(|\.Transaction\(' internal/api internal/dto internal/service`：仅 service/flag.go 三个 Transaction，handler/DTO 无事务或 SQL。
- `git diff --check`：无空白错误（Windows LF/CRLF提示不影响检查）。
- 实际 server curl：用真实 main 编译的 bin/t3a-server.exe、SERVER_ADDR=127.0.0.1:58662，独立随机 smoke schema。`curl.exe` POST 创建→GET 三条件列表→GET详情→PATCH name→PATCH status=false→GET history，状态依次201/200/200/200/200/200；核对默认值、rule_count=0、rules=[]、修改字段、停用值、三条历史最新优先、name old/new，原始 JSON 所有时间均以Z结尾（history 如 `2026-10-09T17:04:35.557031Z`）。最终输出 `CURL SMOKE PASS`。临时服务已停止，schema已 DROP，没有占用5173/8787。
- 清理后 psql 查询 public 三表计数返回 `4|7|4`；`pg_namespace WHERE nspname LIKE 't3a_%'` 返回0，无测试 schema 残留。

## 未尽事项 / 人工决策

- T3a 无未完成项或待人工决策。规则 CRUD/evaluate/batch/copy 属于 T3b，本次未提前实现。
- 没有新建下级 agent；root 后续完成两阶段审查与集成。没有推送、PR、合并或停止主线 dev。
- 并发编辑遵循 D9 的 last-write-wins，无版本冲突检测；事务内历史记录本次读取到的旧值。保留设计规定的已知限制。

提交消息：`feat(T3a): flag CRUD, status and history APIs with transactional audit`。
