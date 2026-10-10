# T5 全量验收：83条功能证据

日期2026-10-10。逐条来源：docs/plan/02-edge-cases.md，全10组实际83条。功能清单83✅/0⚠️/0❌；证据包含实际单测/真实PG集成、真实Go浏览器与新checkout冷重放，测试名指仓库实际测试。不把MSW当作PG持久化证明；前端最终证据由root实际执行并以10张T5-real截图归档。

## 组 1：Flag 创建 / 编辑（T3a）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 1.1 | 同 environment 重复 key | 409 `DUPLICATE_FLAG_KEY`，DB 约束 `uq_flags_env_key` 兜底 | ✅ | API TestCreateAndUniqueEnvironment：同production key第二次409 DUPLICATE_FLAG_KEY；TestRepoMapsRealPostgresUniqueViolations验证真实uq_flags_env_key。 |
| 1.2 | 不同 environment 相同 key | 201 正常创建（题目明确允许） | ✅ | TestCreateAndUniqueEnvironment：a.b-c_d在production/staging各201；seed同key跨环境。 |
| 1.3 | key 大写 / 空格 / 中文 / 以数字开头 / 65 字符 | 400 `INVALID_KEY` | ✅ | T5 root真实HTTP五非法key：大写/空格/中文/数字开头/65字符均400 INVALID_KEY；TestFlagValidationBoundaries对应矩阵亦通过。 |
| 1.4 | key 为 `a.b-c_d` 合通配格式 | 201（正则允许 `.` `_` `-`） | ✅ | TestCreateAndUniqueEnvironment直接创建a.b-c_d为201。 |
| 1.5 | environment = "prod" / 空串 | 400 `INVALID_ENVIRONMENT` | ✅ | T5 root真实HTTP prod/空environment均400 INVALID_ENVIRONMENT；TestValidationAndErrorEnvelope与DTO空环境测试通过。 |
| 1.6 | name 空白 / 101 字符 | 400 `VALIDATION_ERROR` | ✅ | TestValidationAndErrorEnvelope：Unicode空白name及PATCH101个中文字符400 VALIDATION_ERROR。 |
| 1.7 | PATCH 传 key 或 environment 字段 | 400 `IMMUTABLE_FIELD`（不是忽略，是显式拒绝） | ✅ | TestPatchPresenceAndExactDiff：key/environment含同值、空、null均400 IMMUTABLE_FIELD；DTO含转义key、重复key。 |
| 1.8 | PATCH 只改 name，summary 只记 name 变化 | history `flag_updated.changes` 仅含 name | ✅ | TestPatchPresenceAndExactDiff：改name且同值default/description，changes长度1，仅name old/new。 |
| 1.9 | PATCH 传与现值相同的字段 | 该字段不进 changes；全同则不写 history | ✅ | TestPatchPresenceAndExactDiff：全同PATCH与{}均200，history仍2；混合更新仅真实两字段。 |
| 1.10 | 创建 / 修改不存在的 flag id | 404 `FLAG_NOT_FOUND` | ✅ | T5 root真实HTTP PATCH随机不存在UUID404 FLAG_NOT_FOUND；TestValidationAndErrorEnvelope的GET详情/history同样404；POST是创建新资源，不定位既有flag id。 |
| 1.11 | name 带 Unicode（中文）| 201（name 无字符集限制，仅长度） | ✅ | create辅助函数经真实POST使用中文name，201并trim回中文开关；TestFlagValidationBoundaries100中文合法。 |
## 组 2：Flag 启停与列表（T3a / T4a）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 2.1 | PATCH /status enabled=false 后 evaluate | 恒 false，reason=disabled（联动组 4） | ✅ | TestStatusIsIdempotent停用200；TestEvaluateContractAndEngineAgreement与T5smoke验证false/disabled短路。 |
| 2.2 | 重复停用（已 false 再 false） | 200；不写重复 history（值未变化→跳过 history） | ✅ | TestStatusIsIdempotent：第二次false仍200，history保持2。 |
| 2.3 | 列表筛选 env+enabled+q 组合 | 三条件 AND | ✅ | TestListFiltersPaginationAndOrderedDetails：production+true+NEEDLE返回唯一item/total1。 |
| 2.4 | q 命中 name 或 key（大小写不敏感） | ILIKE 语义 | ✅ | 同测试q=NEEDLE大写命中key，q=中文命中name；repo TestFlagCRUDPreservesZeroValuesAndFiltersList验证ILIKE。 |
| 2.5 | page 超出范围 | 返回空 items，不报错；total 正确 | ✅ | TestListFiltersPaginationAndOrderedDetails：page99 items空，total仍3。 |
| 2.6 | page_size=0 / 101 / 非数字 | 取默认 20 / 截断为 100 / 400 | ✅ | 同测试page_size0→20、101→100、nope→400；DTO补重复参数/溢出/负数。 |
| 2.7 | 列表项 rule_count 与实际规则数一致 | COUNT 子查询 | ✅ | 同测试插入三规则，列表rule_count=3，详情priority1/2/3。 |
| 2.8 | 前端 Switch 启停请求失败 | 乐观更新回滚 + 错误 message | ✅ | T5 root停止自有真实Go后点击Switch：请求500，aria-checked=true回滚到先前启用值，显示错误；重启后数据仍启用；T5-real-switch-rollback.png。 |
## 组 3：规则 CRUD（T3b）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 3.1 | priority 与已有规则撞号 | 409 `DUPLICATE_PRIORITY`，DB `uq_rules_flag_priority` 兜底 | ✅ | TestRuleLifecycleScopeAndExactHistory：同priority2 POST409 DUPLICATE_PRIORITY；真实DB唯一约束测试通过。 |
| 3.2 | PATCH 把规则 priority 改成另一规则现有值 | 409 同上 | ✅ | 同测试PATCH priority1撞现有值，409；history仍3。 |
| 3.3 | priority=0 / 负数 / 2.5 / 非整数 | 400 `VALIDATION_ERROR` | ✅ | TestRuleValidation：0/-1/2.5/字符串优先级400，details.field=priority；另含INT上界+1。 |
| 3.4 | operator=equals 但 expected_value 是数组 | 400 `VALIDATION_ERROR`（details 指明 expected_value） | ✅ | TestRuleValidation：equals [1]/[]等400，details.field=expected_value。 |
| 3.5 | operator=in 但 expected_value 是标量 | 400 | ✅ | 同测试in 1/true/pro标量400，details.field=expected_value。 |
| 3.6 | operator=in expected_value=[] 空数组 | 400 | ✅ | 同测试in []400。 |
| 3.7 | expected_value=null / 嵌套对象 / 数组含数组 | 400（D3：只允许标量） | ✅ | 同测试equals/in的null/{}/嵌套对象/[[1]]/[null]/[{}]均400。 |
| 3.8 | expected_value 数字/布尔/字符串标量 | 201 | ✅ | 同测试连续新建42/true/pro标量三规则，各201。 |
| 3.9 | attribute 含前后空格 / 65 字符 | trim 后校验长度；400 | ✅ | TestRuleLifecycleScopeAndExactHistory空格+Unicode空白plan trim后201为plan；TestRuleValidation空白或65中文400。 |
| 3.10 | 向不存在 flag 加规则 | 404 `FLAG_NOT_FOUND` | ✅ | TestRuleLifecycleScopeAndExactHistory：随机不存在flag UUID加规则404 FLAG_NOT_FOUND。 |
| 3.11 | 删除不存在 / 不属于该 flag 的规则 id | 404 `RULE_NOT_FOUND` | ✅ | 同测试PATCH/DELETE另一flag所属rule和随机不存在rid均404 RULE_NOT_FOUND。 |
| 3.12 | 对 disabled flag 增删改规则 | 允许（D4 只影响评估），history 照记 | ✅ | 同测试disabled后创建vip规则201、更新priority200、删除204，历史仍记录完整摘要。 |
| 3.13 | ★ 原子性：构造 history 写入失败，验证规则未落库（或反之） | 事务回滚，两者都不存在 —— API 集成测试用例 | ✅ | TestRuleHistoryFailuresRollbackCRUD：隔离PG CHECK(false)失败注入，create/update/delete各500且原rule/两条history保持；真实事务回滚。 |
| 3.14 | rule_updated summary 只记变化字段（如仅 roll_value） | changes 精确 | ✅ | TestRuleLifecycleScopeAndExactHistory：仅roll_value true→false，changes严格仅该字段；同值{}不新增历史。 |
## 组 4：评估引擎（T2 单测全覆盖）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 4.1 | disabled 且规则会命中 | false / reason=disabled / trace 仅 1 条短路记录（D4） | ✅ | TestEvaluate/4.1_disabled_short_circuit及API引擎一致性测试：false/disabled/单条short_circuited_by_disabled_flag。 |
| 4.2 | enabled + 无规则 | default_value / reason=default / trace=[] | ✅ | TestEvaluate/4.2_empty_rules_default_false与4.2_nil_rules_default_true：default与trace[]；API no-rules200。 |
| 4.3 | 全部规则跳过（属性全缺失） | default_value / trace 每条 attribute_missing | ✅ | TestEvaluate/4.3_all_attributes_missing：每规则attribute_missing，最后default。 |
| 4.4 | 第 2 条命中，后面还有 3 条 | trace 只含前 2 条（命中即停） | ✅ | TestEvaluate/4.4_second_hit_stops_before_three_remaining_rules：仅两条Trace；API同输入一致。 |
| 4.5 | 规则乱序传入（priority 3,1,2） | 按 1→2→3 评估 | ✅ | TestEvaluate/4.5_priority_3_1_2_sorted_without_mutation：1→2→3，原slice不变。 |
| 4.6 | equals：attrs.plan="pro" vs expected "pro" | 命中 | ✅ | TestEvaluate/4.6_equals_string_hit：pro字符串命中；API+T5smoke同值hit。 |
| 4.7 | equals：attrs.level=42 vs expected 42 | 命中（双侧归一 float64） | ✅ | TestEvaluate/4.7_equals_json_float64_vs_int_42与反向：双侧数字归一命中；APIlevel42.0命中。 |
| 4.8 | equals：attrs.vip=true vs expected true | 命中 | ✅ | TestEvaluate/4.8_equals_bool_hit；API vip=true命中。 |
| 4.9 | equals："1" vs 1 / true vs "true" | 不命中，cause=type_mismatch（D3） | ✅ | TestEvaluate/4.9_equals_string_1_vs_number_1与bool/string_true：不命中type_mismatch。 |
| 4.10 | equals 值不等（"pro" vs "team"） | cause=value_not_equal | ✅ | TestEvaluate/4.10_equals_same_type_different_value：value_not_equal。 |
| 4.11 | in：命中列表某元素 / 全不在 | matched / value_not_in_list | ✅ | TestEvaluate/4.11_in_hit_returns_false_roll_value、in_miss、single_element_hit：matched或value_not_in_list。 |
| 4.12 | in 列表混合类型 ["pro", 42]，attrs 42 | 命中（仅同类型比较） | ✅ | TestEvaluate/4.12_in_mixed_types_numeric_hit与strict_string_42_miss：只同类型命中。 |
| 4.13 | attrs 无规则所指属性 | 该规则 skip 继续（D5） | ✅ | TestEvaluate/4.13_missing_attribute_continues_to_hit及nil_user_attrs：skip并继续，无panic。 |
| 4.14 | attrs 值是嵌套对象，规则匹配其顶层 key | 对象≠标量 → 不命中（in→not_in_list, equals→type_mismatch） | ✅ | TestEvaluate/4.14_nested_object_equals_scalar/in_scalar_list及nested_array：equals type_mismatch，in value_not_in_list。 |
| 4.15 | default_value=true 全 miss | 返回 true / reason=default | ✅ | TestEvaluate/4.15_default_true_all_miss；API改default=true后{}返回true。 |
## 组 5：评估 API（T3b / T4b）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 5.1 | flag_key+env 查不到 | 404 `FLAG_NOT_FOUND` | ✅ | TestEvaluateRequestValidationBeforeLookup：合法object但missing-flag404 FLAG_NOT_FOUND。 |
| 5.2 | body 非法 JSON（少引号/尾逗号） | 400 `INVALID_JSON`，message 含位置 | ✅ | 同测试少闭合/尾逗号400 INVALID_JSON，message含byte；DTO同样位置校验。 |
| 5.3 | user_attributes 是数组 / 字符串 / 数字 | 400 `INVALID_USER_ATTRIBUTES` | ✅ | 同测试null/[]/string/number/bool与缺失属性400 INVALID_USER_ATTRIBUTES，先于资源查找。 |
| 5.4 | user_attributes 含嵌套对象/数组值 | 200（合法），规则只匹配顶层 | ✅ | TestEvaluateContractAndEngineAgreement：嵌套object/array/null属性200，与引擎顶层语义逐Trace一致。 |
| 5.5 | environment 非法 | 400 | ✅ | TestEvaluateRequestValidationBeforeLookup：prod400 INVALID_ENVIRONMENT。 |
| 5.6 | 评估结果与直接调 eval 引擎一致（同输入） | 契约一致性 | ✅ | TestEvaluateContractAndEngineAgreement九组真实API结果value/reason/Trace/rule_id/priority逐项对照eval.Evaluate。 |
## 组 6：批量模拟（T3b / T4b）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 6.1 | 5 组 attrs，2 组命中 #2 规则、1 组默认、1 组属性缺失跳过 | summary 计数正确，hit_rules 按次数排序 | ✅ | TestBatchStatisticsLimitsAndQueryCount真实PG五组total5/true3/false2/default1/#2hit2；T5真实seed浏览器另五组3true/2false/default2/5，#2两次与#1一次降序，展开Trace可见。 |
| 6.2 | list 为空数组 | 400 `VALIDATION_ERROR` | ✅ | 同测试list=[]400 VALIDATION_ERROR。 |
| 6.3 | list > 100 | 400 `BATCH_LIMIT_EXCEEDED` | ✅ | 同测试101组400 BATCH_LIMIT_EXCEEDED，100组200。 |
| 6.4 | 第 3 组非法（非 object） | 整体 400，details 标注 index=2（预演场景要全对才跑） | ✅ | 同测试第三组null/[]/string/number/bool整体400，details.index2且查库次数0。 |
| 6.5 | flag disabled + 批量 | 全部 false + reason=disabled，summary.default_count=0、true_count=0 | ✅ | 同测试disabled五组false/disabled，true_count0/default_count0/hit_rules空，仅flags一次查询；T5smoke两组同样通过。 |
| 6.6 | 前端 textarea 每行一个 JSON，含空行/注释行 | 前端先本地校验：空行忽略、坏行标行号提示 | ✅ | T5 root真实Go浏览器：{}、空行、注释/坏JSON标原始第3行且开始模拟禁用；五组实际3true/2false/default2/5，空行忽略；T4b实际parser映射与100上限亦通过。 |
## 组 7：操作历史（T3a / T4b）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 7.1 | 8 种 action 各触发一次 | summary 结构符合 01-design §8-C schema | ✅ | TestCreateAndUniqueEnvironment、TestPatchPresenceAndExactDiff、TestStatusIsIdempotent、TestRuleLifecycleScopeAndExactHistory、TestCopyPreservesSourceAndPriorities组合触发8action并断言对应summary schema。 |
| 7.2 | history 按 created_at DESC 分页 | 最新在前 | ✅ | TestStatusIsIdempotent与repo TestHistoryListReturnsLatestFirstAndCorrectTotal验证最新优先分页/total；T5真实Go历史初始5/6，加载更多6/6，删除/更新在前。 |
| 7.3 | flag_updated changes 只含实际变化字段 | 见 1.8 / 3.14 | ✅ | TestPatchPresenceAndExactDiff与TestRuleLifecycleScopeAndExactHistory精确changes，JSON42/42.0同值不写历史。 |
| 7.4 | rule_deleted summary 记录被删规则的完整摘要 | 删除后仍可追溯 | ✅ | TestRuleLifecycleScopeAndExactHistory删除后summary保留priority4/plan/in/pro/false全部五字段。 |
| 7.5 | DiffView 渲染 old→new（删除线红/新值绿） | 布尔值渲染为 Tag | ✅ | T5 root真实Go历史页：computedStyle旧值rgb(207,19,34)+line-through，新值rgb(35,120,4)，4个boolean Tag；截图T5-real-history.png。 |
## 组 8：环境复制（T3b stretch / T4b）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 8.1 | staging→production 成功 | 201，新 flag enabled=false，规则 priority 全保持 | ✅ | TestCopyPreservesSourceAndPriorities staging→production201保持priority2/7；T5真实UI development→staging201，新iddb418f86-a3da-4011-a258-891a89b35eef停用，priority[1,3]保持；复制确认明确停用。 |
| 8.2 | 目标环境已有同 key | 409 `DUPLICATE_FLAG_KEY` | ✅ | 后端TestCopyPreservesSourceAndPriorities目标重复409；T5真实staging→development409友好反馈，取消/选择目标可恢复。 |
| 8.3 | target == source | 400 `COPY_TARGET_CONFLICT` | ✅ | 同测试target=staging/source400 COPY_TARGET_CONFLICT。 |
| 8.4 | 复制后新 flag 的 history | flag_copied，summary 含 source 信息 | ✅ | 后端TestCopyPreservesSourceAndPriorities精确flag_copied summary；T5真实新staging详情history含source development和源id。 |
| 8.5 | 复制后原 flag 不受影响 | 源 flag 规则/状态不变 | ✅ | 同后端测试copy前后源详情/history深相等，失败注入全部回滚；T5真实复制后源规则priority[1,3]/状态不变。 |
## 组 9：前端交互通用（T4a / T4b）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 9.1 | 提交按钮重复点击 | loading 期间禁用 | ✅ | T5 root仅锁自己测试Flag行，真实规则INSERT暂阻塞：confirm/cancel/attribute的isEnabled=false、保存中弹窗保留；截图T5-real-pending.png；T4a/T4b另有创建/单次/batch防重复禁用实证。 |
| 9.2 | 409 错误（key/priority 撞号） | 内联定位到对应表单字段 + 解释文案 | ✅ | T5 root真实Go浏览器：同环境key409定位Key字段，规则priority409定位优先级字段，输入保留；后端TestCreateAndUniqueEnvironment/TestRuleLifecycleScopeAndExactHistory均通过。 |
| 9.3 | JSON 编辑器非法输入 | 不发请求，编辑器下方标错 | ✅ | T5 root真实Go浏览器：重复token坏JSON准确标第3行并禁用提交；格式化成功；equals null本地拒绝。T4b实际源码grammar诊断/native fallback/finite回归均有红绿证据。 |
| 9.4 | 各列表/表单空态 | 引导文案而非白屏 | ✅ | T5 root真实Go搜索never-exists-t5-empty，显示新建Flag引导而非白屏；新Flag初始0规则显示添加引导；T5-real-empty.png；T4b无Trace空态已有证据。 |
| 9.5 | 网络断开/后端未起 | 友好错误，可重试 | ✅ | T5 root停止Go后重载：列表失败Alert/友好message+重试；重起Go后点重试恢复5Flags且数据保留；404输入可修正。 |
| 9.6 | 删除规则二次确认 | Popconfirm | ✅ | T5 root真实Go：Popconfirm确认前规则2条，确认后1条；history加载更多5/6→6/6，删除后仍完整tier摘要；T5-real-history.png。 |
| 9.7 | key 输入即时正则校验 | 与后端 `^[a-z][a-z0-9._-]{1,63}$` 完全一致 | ✅ | T5 root真实Go浏览器非法key/required内联拒绝；T4a实际浏览器Bad-Key/空格/1bad即时校验；前后端同一^[a-z][a-z0-9._-]{1,63}$，DTO全非法矩阵通过。 |
## 组 10：复现与工程（T0 / T5）

| # | 场景 | 预期 | 判定 | 实际证据 |
| --- | --- | --- | --- | --- |
| 10.1 | `docker compose down -v && up` + migrate down/up | 干净可逆，无残留对象 | ✅ | T5 root新checkout88fcb37实际docker compose down -v/up，migrate up/down/up全部成功；down后三张业务表/索引/序列关系计数0；共享pgcrypto/schema_migrations按设计保留。 |
| 10.2 | seed 重复执行 | 幂等，不产生重复 flag / 重复 history | ✅ | T5 root fresh PG up/down/up后双次seed均seed complete，实际4Flag/7Rule/4History不重复；TestSeedIdempotentPreservesExistingFlagsAndMissingRules亦通过。 |
| 10.3 | `make dev` 全新 clone 一条命令起全套 | README 步骤可复现 | ✅ | T5 root新checkout C:/Users/77230/FlagForge/.worktrees/reproduction/FlagForge，HEAD88fcb37、新coldGOCACHE；按README前置安装后bash scripts/dev.sh一次起Compose/migrate/seed/Go8787/Vite5173，health/HTML/真实proxy列表均200，无文档外步骤。 |
| 10.4 | `go test ./...` 全绿 | eval 单测 + API 集成测试 | ✅ | T5 go test ./... -count=1与显式真PG go test -tags=integration ./... -count=1均exit0，api9.685s/dto0.260s/eval0.223s/repo0.640s/seed0.494s。 |
| 10.5 | .gitignore 覆盖 .env / node_modules / 构建产物 | 仓库干净 | ✅ | .gitignore覆盖.env/.env.local/.env.*.local/node_modules/dist/backend/bin/日志/.worktrees；env开发/生产默认开关作为非敏感配置跟踪；未跟踪数据卷。 |
| 10.6 | smoke.sh 冒烟序列全过 | 创建→加规则→评估→启停→历史 全链路 | ✅ | T5 root新checkout/重建PG的scripts/smoke.sh十步HTTP/JSON全PASS exit0，key smoke-mv1rlcrt-77c64d86、id853b7291-ded3-42d3-9851-affdec9be17b；executor隔离schema正向PASS且HTTP/JSON失败非零。 |

## 限制对应

83条功能清单没有❌/⚠️。额外可选体验的OS剪贴板读回仍⚠️：按钮/命令生成/转义已有实证，实际读回为空，未声称完整剪贴板写入。额外npm audit报告4项（3moderate/1high），没有声称零问题。两者及无前端框架、chunk警告、float64、last-write-wins、admin、顶层属性、无Flag删除等边界均对应README§8，不混入83条功能计数。正式提交时限不可仅由Git验证。

## 统计

| 组 | ✅ | ⚠️ | ❌ |
| --- | --- | --- | --- |
| 1 创建/编辑 | 11 | 0 | 0 |
| 2 启停/列表 | 8 | 0 | 0 |
| 3 规则CRUD | 14 | 0 | 0 |
| 4 引擎 | 15 | 0 | 0 |
| 5 评估API | 6 | 0 | 0 |
| 6 批量 | 6 | 0 | 0 |
| 7 历史 | 5 | 0 | 0 |
| 8 环境复制 | 5 | 0 | 0 |
| 9 前端通用 | 7 | 0 | 0 |
| 10 复现工程 | 6 | 0 | 0 |
| 总计 | 83 | 0 | 0 |
