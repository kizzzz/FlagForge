# T3b 执行报告

执行日期：2026-10-10（Asia/Shanghai）
worktree：`C:/Users/77230/FlagForge/.worktrees/t3b`；分支：`codex/t3b-rules-evaluation`；基线：`a20dd59`。
状态：**DONE，核心及 copy stretch 实现与自验完成，交由 root 两阶段审查。**

## 环境探测结果

- Windows / PowerShell；Go 使用 `C:/Program Files/Go/bin/go.exe`，项目保持 Go 1.22 和既有依赖。
- PostgreSQL 使用现有 Compose 项目 flagforge、root docker-compose.yml、localhost:55432；未创建第二个 Compose、未删除卷。
- 主线 5173 / 8787 与前端独立 5174 均未操作。真实服务 smoke 使用临时空闲端口 58020，进程 14804 已停止。

## 做了什么

- 阅读完整 T3b 提示词、design §5 D2–D5 / §6.2–6.5 / §8 A–D、edge 组3/4/5/6/8、T2/T3a 报告。完整原文与 root 委派及补充澄清归档至 T3b-prompt.md。
- 增加规则 POST/PATCH/DELETE 的 DTO、service、handler；新增单次 evaluate、batch 模拟；核心全量集成测试 PASS 后实现环境 copy。
- 规则校验 equals 标量 / in 非空全标量数组、必填 roll_value 指针、attribute Unicode trim 与1–64字符、priority 为1–2147483647整数。显式 null、错误类型与 unknown 字段返回具体 field 的400；重复 priority 创建和更新均复用 repo.DuplicatePriority 映射409。
- 规则由 repo.GetByID 的 flag_id + id 条件限定归属；disabled flag 的规则操作正常允许。三种规则操作的业务与 history 在一个 PG 事务内，沿用 WriteHistory，不增加内部事务。
- 单次响应包含 flag_key/environment/enabled/value/reason/trace/evaluated_at；trace 使用引擎结果、命中后不补后续规则；disabled 仅 matched/cause。rules 返回时间与 copy flag 返回时间显式 UTC。
- copy 新建 enabled=false flag，复制全部规则并保留 priority/gaps/expected_value/roll_value，写入一条 flag_copied history（flag摘要及source flag_id/environment），源内容与历史不变。目标同源400、目标已有key409、源不存在404。
- 新增两份真实 PG API 集成测试文件，复用 T3a setupAPI 随机 schema，并在 Cleanup 中关闭连接 / DROP schema。未修改 eval/repo/model/迁移/seed/frontend/README/scripts/main/go.mod/go.sum。

## 关键决策

1. **双侧归一**：DB jsonb 经 encoding/json 解码后，equals 标量调用 eval.Normalize，in 数组逐项调用；请求属性复制为新 map，各顶层值也调用 eval.Normalize。嵌套对象/数组保留原值，仍交纯引擎按严格类型和顶层属性语义处理。没有改引擎或在service自行比較。summary 用 json.Marshal 已归一的 expected，生成紧凑 JSON，如 `plan equals "pro" → true`。
2. **先校验后查库**：DecodeObject 负责语法错误位置；DecodeEvaluate 在通用 DecodeFields 前专门解析 user_attributes，missing/null/数组/字符串/数字/布尔均 INVALID_USER_ATTRIBUTES。Validate 在 service 也执行，所有合法性检查先于查 flag。评估非法 key 使用 VALIDATION_ERROR、details.field=flag_key；copy环境校验定位 target_environment。
3. **batch整体失败**：先把 user_attributes_list 解析为 RawMessage数组，检查1–100长度，并逐组验证 object；非法组返回整批400、details.index 为0起始，同时不查数据库。合法批次 load 一次 flag + 一次规则，循环调引擎。真实 GORM 查询回调计数断言仅 flags/rules 两次；disabled仅一次 flags。统计 true/false 与 default_count；disabled不进入default_count/hit_rules，不增加契约之外disabled_count字段。hit_rules仅rule_hit且包含roll_value=false命中，按count降序、priority升序、rule_id升序稳定排序。
4. **部分更新与历史diff**：PATCH保留各字段存在性；把传入字段合并到当前规则后完整校验 operator/expected，防止仅改operator形成非法组合。比较解析后的 JSON 值，空白或42→42.0不写假历史，数组顺序仍有意义。布尔false通过指针和map更新保留；精确changes只含实际变化字段；无diff不更新updated_at、不写history。rule_deleted记录五个规则摘要字段；rule_updated包含rule_id、更新后priority及changes。
5. **copy stretch已完成**：先完成规则/evaluate/batch全量GREEN，再独立写copy真实RED测试并实现。事务包住新flag、全部rules、flag_copied；中间规则失败或最后history失败均整笔回滚，没有提前提交或伪造历史。

## 红 → 绿证据

- 核心先新增6组API测试，`go test ./internal/api -tags=integration -run 'TestRule|TestEvaluate|TestBatch' -count=1 -v` exit1；请求真实PG创建flag成功201，随后缺失规则路由 expected201/400、actual404，evaluate/batch expected200、actual404。失败源自功能缺失，没有编译或依赖错误。priority=2147483648作为初始RED矩阵的一项，GREEN后400定位priority；PATCH同值也覆盖。
- 核心实现后 `go test ./... -tags=integration -count=1` exit0，api9.898s、dto0.931s、eval0.163s、repo8.256s、seed8.032s。至此才开始copy。
- copy测试先运行 `go test ./internal/api -tags=integration -run TestCopy -count=1 -v` exit1，目标同源期望400实际404、两种真实失败注入期望500实际404。实现后同命令exit0、api7.233s：规则第二条CHECK真实23514与history CHECK真实23514分别触发，断言flag总数仍1、rules仍2、源完整内容和历史不变。
- 规则CRUD history失败注入 `CHECK(false) NOT VALID`：create/update/delete均500 INTERNAL_ERROR，rule仍为原ID/priority/roll_value，历史条数未变；数据库23514只记录日志，响应没有内部细节。
- priority定向回归证明：临时移除DTO的INT上限后，`go test ./internal/api -tags=integration -run 'TestRuleValidation/.*2147483648' -count=1 -v` exit1，真实pgx报 `2147483648 is greater than maximum value for int4`，API actual500/expected400；finally恢复原文件后同命令exit0，API400且定位priority。临时修改未保留。

## 验收自验清单

- [x] 最终 `go test ./... -tags=integration -count=1` exit0：api11.620s、dto1.015s、eval0.165s、repo7.853s、seed1.224s。
- [x] `go test ./... -count=1` exit0，普通dto/eval/repo测试通过；不带integration不连接PG。
- [x] `go vet ./...`、`go build ./...`、`go build -o bin/t3b-server.exe ./cmd/server` exit0，无诊断。
- [x] 规则覆盖撞号创建+更新409、全部非法expected形态、priority0/负/小数/字符串/超INT、attributetrim/空/65字符、归属404、disabled CRUD、仅roll_value精确changes、仅operator合并非法、null/unknown、同值/空PATCH、42与42.0不新增历史及CRUD真实回滚。
- [x] evaluate对照既有纯引擎：字符串/数值/布尔命中、类型不同、属性缺失、对象/数组/null属性、default true/无规则、命中停止、summary、UTC时间、404/JSON位置/属性400先于404、非法环境/flag_key定位。
- [x] batch混合命中统计、false roll命中统计、count并列稳定排序、空/100/101组、非数组list、任一组非法index=2、非法组不查库、enabled两次查询/disabled一次查询、disabled全部false/default_count0/hit_rules空。
- [x] copy组8全部，以及空规则复制、description/default保留、源规则ID与状态不变、目标独立ID、flag_copied单历史、rule/history失败全回滚。
- [x] 真实main server curl三次HTTP200：§6.3 checkout-redesign-2026/CN/pro为true、priority2、summary精确、trace仅2条；seed billing-portal为false/disabled、trace仅matched/cause；batch4组为true3/false1/default1，数值user_id9001命中。输出 `CURL SMOKE PASS port=58020 process=14804`，finally已停止自己server。
- [x] curl仅对public seed读操作；测试全部随机schema。最终psql public计数 `4|7|4`，t3a_%/t3b_%临时schema数0。
- [x] `rg -n '\.Begin\(|\.Transaction\(' backend/internal/api backend/internal/dto backend/internal/service`：事务仅在service；新增rule三处/copy一处。`rg -n 'Normalize|map\[string\]any\(' backend/internal/service/evaluator.go`：Normalize两侧调用可见，裸转换无匹配。
- [x] `git diff --check` 无空白错误；gofmt已应用。Windows换行提示仅工具提示，既有文件的换行变动已恢复。

## 给后续任务的公共接口

```go
// internal/dto
CreateRule / UpdateRule // 指针priority/attribute/operator/roll_value + RawMessage expected_value
DecodeCreateRule(body []byte) (CreateRule,error)
DecodeUpdateRule(body []byte) (UpdateRule,error)
ValidateExpected(operator string,raw json.RawMessage) error
EvaluateRequest / BatchRequest / EvaluateResponse / BatchResponse
DecodeEvaluate(body []byte) (EvaluateRequest,error)
DecodeBatch(body []byte) (BatchRequest,error)
CopyFlag{TargetEnvironment string}
DecodeCopyFlag(body []byte) (CopyFlag,error)
// 各请求类型均有 Validate() error。

// internal/service
NewRuleService(db *gorm.DB) *RuleService
(*RuleService).Create(ctx context.Context,flagID string,r dto.CreateRule) (*model.Rule,error)
(*RuleService).Update(ctx context.Context,flagID,id string,r dto.UpdateRule) (*model.Rule,error)
(*RuleService).Delete(ctx context.Context,flagID,id string) error
NewEvaluatorService(db *gorm.DB) *EvaluatorService
(*EvaluatorService).Evaluate(ctx context.Context,r dto.EvaluateRequest) (*dto.EvaluateResponse,error)
(*EvaluatorService).Batch(ctx context.Context,r dto.BatchRequest) (*dto.BatchResponse,error)
(*FlagService).Copy(ctx context.Context,id string,r dto.CopyFlag) (*model.Flag,error)

// internal/api
NewRuleHandler(s *service.RuleService) *RuleHandler
NewEvaluatorHandler(s *service.EvaluatorService) *EvaluatorHandler
// RegisterRoutes 已接入上述六个端点；既有接口不变。
```

## 未尽事项 / 人工决策

- 本任务无未完成项；copy stretch已完成。无需新增依赖、数据库迁移或修复T2引擎。
- 沿用D9并发编辑last-write-wins，无乐观锁；属性只支持顶层key。数字沿用encoding/json float64与T2 Normalize口径。
- 没有创建下级agent，没有推送/PR/合并；保留worktree与分支供root两阶段审查。临时服务已停止、测试schema已清理，主线dev仍由root管理。

提交消息：`feat(T3b): rules CRUD, evaluation, batch simulation and env-copy APIs`。
