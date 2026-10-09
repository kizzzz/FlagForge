# T3b 原始提示词归档

执行日期：2026-10-10（Asia/Shanghai）
来源：docs/plan/prompts/T3b-rules-eval-api.md；下方完整保留任务原文。
worktree：C:/Users/77230/FlagForge/.worktrees/t3b；分支：codex/t3b-rules-evaluation；基线：a20dd59。
# 任务 T3b：规则 CRUD / 评估 / 批量 / 复制 API

## 你的角色

你是 FlagForge 的后端业务工程师（第二阶段）：规则管理 API、评估端点（本项目对外的心脏）、
批量模拟与环境复制。你消费 T2 的纯函数引擎与 T3a 立下的基建模式。

## 必读（动手前读完）

1. `docs/plan/01-design.md` §5 决策 D2/D3/D4/D5、§6.2~§6.5（契约）、§8-A/B/D（亮点规格）
2. `docs/plan/02-edge-cases.md` 组 3、组 4（对照语义）、组 5、组 6、组 8
3. `docs/ai-logs/T2-report.md`（引擎契约与 Normalize 用法）、`T3a-report.md`（错误信封/事务模式）

## 交付物

### 1. 规则 CRUD `service/rule.go` + handlers

- expected_value 校验（D3，组 3.4~3.7，**前后端同口径**）：
  - `equals` → JSON 标量（string|number|bool）；null/对象/数组 → 400
  - `in` → 非空数组且元素全为标量；否则 400
  - 错误 details 指明字段 `expected_value` 与原因
- priority ≥1 整数；撞号 → 409 `DUPLICATE_PRIORITY`（复用 repo 映射）
- history：`rule_created` / `rule_updated`（仅变化字段，同 T3a diff 模式）/ `rule_deleted`（被删规则完整摘要，组 7.4）
- 事务原子性同 T3a 标准

### 2. 评估 `service/evaluator.go` + handler `POST /api/v1/evaluate`

组装流程（顺序即语义）：

```
1. 按 flag_key + environment 查 flag → 查不到 404 FLAG_NOT_FOUND
2. body 层：JSON 语法错 → 400 INVALID_JSON（message 含解析错误位置）；
   user_attributes 非 object → 400 INVALID_USER_ATTRIBUTES（先于 404 校验请求合法性）
3. 查规则（priority 升序）→ 组装 eval.Input：
   - DB jsonb 侧 expected_value 经 eval.Normalize 归一（T2 导出的函数，双侧同口径）
   - 请求侧 user_attributes 同样归一
4. eval.Evaluate → 映射响应：value / reason{type, rule_id, priority, summary} / trace / evaluated_at
5. reason.summary 生成：`{attribute} {operator} {expected 的紧凑 JSON} → {roll_value}`
   例：`plan equals "pro" → true`、`country in ["US","CA","GB"] → true`
```

### 3. 批量 `POST /api/v1/evaluate/batch`

- 1~100 组（组 6.2/6.3）；任一组非法 → **整体 400**，details 标注 `index`（组 6.4）
- 查一次 flag + 一次规则，循环调引擎（不逐组查库）
- summary 按 §6.4：true/false_count、default_count、hit_rules（按 count 降序）
- disabled flag 批量：全 false + reason=disabled，summary 计数正确（组 6.5）

### 4. 环境复制 `POST /api/v1/flags/:id/copy`（stretch：核心全做完再做）

- 校验：target≠source（400 COPY_TARGET_CONFLICT）；目标 env 同 key → 409 DUPLICATE_FLAG_KEY
- 单事务：建新 flag（**enabled=false** 安全默认）→ 复制全部规则（priority 保持）→ history `flag_copied`（§8-C）
- 时间紧时跳过本节并在报告注明 —— 优先级声明见 00-README 裁剪顺序

### 5. 集成测试补充

- 规则：撞号 409（创建+改优先级两种路径，组 3.1/3.2）、expected_value 四种非法形态（3.4~3.7）、
  不属于该 flag 的 rule id → 404（3.11）
- evaluate：命中 / 默认 / 停用短路 / 404 / INVALID_JSON / INVALID_USER_ATTRIBUTES（组 5 全部）
- 批量：混合命中统计正确性（组 6.1）、>100 限流（6.3）、index 标注（6.4）
- 原子性：rule 写入成功但 history 失败 → 两者皆无（复用 T3a 注入手法）

## 本任务已知陷阱

- **双侧归一**：DB jsonb 取出的数字与请求 JSON 解析的数字形态不同 —— 忘记 Normalize 就会出现
  「单测全绿、API 全挂」的经典事故（T2 的 `DeepEqualNormalized` 就是为这一步准备的）
- evaluate 的请求校验顺序：先 body 合法性再查库（404 之前先 400），契约已定，别颠倒
- batch 的 hit_rules 统计口径：按「命中规则」计数，default/disabled 不进 hit_rules
- trace 中命中之后的规则不出现（引擎已保证，别在 service 层画蛇添足补全）

## 验收标准

- [ ] `go test ./... -tags=integration` 全绿（含组 3/5/6/8 API 用例）
- [ ] curl 演示 01-design §6.3 的示例请求，响应结构与契约逐字段一致（trace 顺序、cause 枚举）
- [ ] 停用的 billing-portal flag（seed 数据）evaluate → false + disabled + 短路 trace
- [ ] `grep -rn "map\[string\]any(" backend/internal/service/evaluator.go` —— 确认无裸比较绕过 Normalize

## 硬约束

- 引擎只调不改：发现引擎 bug → 报告上报，由人工决定是否回修 T2（不要顺手改 eval 包破坏其测试）
- copy 是 stretch：核心（规则/评估/批量）验收不过时禁止开工 copy

## 完成后必做

写 `docs/ai-logs/T3b-report.md`（模板同 T0；「关键决策」必须包含：双侧归一落地方式、
batch 整体失败语义的实现、copy 是否完成及原因），
然后 `git commit -m "feat(T3b): rules CRUD, evaluation, batch simulation and env-copy APIs"`。

## root 委派上下文与后续澄清（完整）

实现FlagForge T3b，用户授权补齐完整题面，计划已批准。只在 C:/Users/77230/FlagForge/.worktrees/t3b 工作，分支codex/t3b-rules-evaluation，基线a20dd59（T3a已规格与质量PASS、root完整测试PASS并合并）。阅读 docs/plan/prompts/T3b-rules-eval-api.md 严格执行全文及design §5D2~D5/§6.2~6.5/§8A/B/D、edge组3/4/5/6/8、T2-report/T3a-report。
交付规则CRUD DTO+service+handlers、单次evaluate、batch统计，核心全绿后再做copy stretch（前端T4b已做好client/mock与确认UI）。规则equals只string|number|bool禁止null/数组/对象，in非空全标量数组；priority>=1整数；API400具体field，重复创建/改priority映射repo.DuplicatePriority→409；scope rule to flag，disabled允许规则操作；rule_created/updated精确changes/deleted完整rule摘要同业务PG事务。修改无变化不历史，false不得忽略。先TDD真实PG失败再实现。
Evaluate请求准确 flag_key/environment/user_attributes，先body/属性object校验再查flag，错误400先于不存在404。调用既有纯eval引擎（不可修改），DB JSON和请求两侧数字归一，disabled不参与规则评估（可直接短路不加载规则）。响应按§6.3 value/reason/trace/evaluated_at，trace命中后不加规则；short-circuit仅cause/matched无rule_id；summary紧凑JSON示例 plan equals "pro" → true。batch user_attributes_list1~100，任一非法整批400+details.index0起，查一次flag一次rules，不每组查；统计default/disabled分别计、hit_rules只rule_hit按count降序（tie deterministic）。copy核心PASS后实现：目标不同源、撞key409，新flag enabledfalse，规则全copypriority保留，新历史flag_copied含source，单事务原子。
公共接口T3a-report有精确签名：api.NewRouter/RegisterRoutes/ReadBody/HandleError/Respond*；业务错误internal/apperror.Error与13codes；DTO DecodeObject/DecodeFields/ValidateKey/Env/ID/ParsePageQuery；service.WriteHistory(ctx,tx,...)。重用已有基建，错误PG23505只repo映射。DecodeFields拒绝null/unknown，属性object或expected专门错误先处理RawMessage。服务回读timestamp可能本地zone，规则/copy所有响应created_at/updated_at显式.UTC；flagUTC同service包可用。NowFuncUTC不够，T3a已复现并修复。
环境PowerShell，Go C:/Program Files/Go/bin；DATABASE_URL postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable；DockerCLI %LOCALAPPDATA%/Programs/DockerDesktop/resources/bin/docker.exe，单Compose项目flagforge/rootyaml。主线5173/8787仍运行（旧health-only binary会由root最后重启），不能停止；frontendagent5174独立。APItests可复用api包 setupAPI() 隔离schema（private helper同包），history失败用隔离真实CHECK，不改public4/7/4、不删除volume、不启动另一个Compose。curl可用SERVER_ADDR自有空闲端口，不改默认8787。不要改frontend/README/scripts/T2 eval/迁移。保持go1.22/依赖。
覆盖edge3/5/6/8，包括rule创建和更新重复409、expected非法全类型、rule归属404、rule历史写失败CRUD回滚、匹配/默认/disabled/JSON错误位置/非法属性先400/404、batch统计/空/>100/index/disabled；copy若做测试规则复制且源不变、失败全rollback。不要创建子agents，root负责两阶段审查。
完成归档全文T3b-prompt/report（归一、batch整批失败、copy是否完成、公共接口、红绿真实证据、问题限制），按提示词commit，停止自己临时server、schema清理，回报状态/SHA/全部验证/修改清单。

后续消息1：严格按§6.4，不新增disabled_count。补充“分别计”指disabled不能误进default_count或hit_rules，disabled批量false_count=total/default_count=0即可。保持现有契约。PATCH先合并operator/expected再验证方向正确，请覆盖仅改operator、显式null/false/priority重复以及无变化不写history。评估flag_key验证details字段请是flag_key，DTO ValidateKey可把其key错误定位映射到请求字段；§6.3如非法格式按VALIDATION_ERROR即可，避免前端把Key错误定位到JSON编辑器。

后续消息2：核心GREEN已收到，按契约省略disabled_count正确。收尾请检查priority与PG INT兼容：Go int可接受>2^31-1，但PG会22003；DTO应对超出INT范围给400 VALIDATION_ERROR，不应由DB兜成500（至少2147483648定向红绿）。expected_value diff请基于解析后的实际JSON值，避免仅空白/数字表示1 vs1.0变化写假history；无需改变数组顺序语义。copy做完请及时提交供审查，T4b在修两项规格P2。