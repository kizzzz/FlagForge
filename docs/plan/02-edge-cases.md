# FlagForge 边界情况清单 v1.0

> 兼作 T5 最终验收清单。每条 = 场景 + 预期行为。executor 开发时对照自己负责的组；
> T5 逐条验证并打勾。「定义」列的编号引用 01-design.md 的设计决策。

## 组 1：Flag 创建 / 编辑（T3a）

| # | 场景 | 预期 |
|---|------|------|
| 1.1 | 同 environment 重复 key | 409 `DUPLICATE_FLAG_KEY`，DB 约束 `uq_flags_env_key` 兜底 |
| 1.2 | 不同 environment 相同 key | 201 正常创建（题目明确允许） |
| 1.3 | key 大写 / 空格 / 中文 / 以数字开头 / 65 字符 | 400 `INVALID_KEY` |
| 1.4 | key 为 `a.b-c_d` 合通配格式 | 201（正则允许 `.` `_` `-`） |
| 1.5 | environment = "prod" / 空串 | 400 `INVALID_ENVIRONMENT` |
| 1.6 | name 空白 / 101 字符 | 400 `VALIDATION_ERROR` |
| 1.7 | PATCH 传 key 或 environment 字段 | 400 `IMMUTABLE_FIELD`（不是忽略，是显式拒绝） |
| 1.8 | PATCH 只改 name，summary 只记 name 变化 | history `flag_updated.changes` 仅含 name |
| 1.9 | PATCH 传与现值相同的字段 | 该字段不进 changes；全同则不写 history |
| 1.10 | 创建 / 修改不存在的 flag id | 404 `FLAG_NOT_FOUND` |
| 1.11 | name 带 Unicode（中文）| 201（name 无字符集限制，仅长度） |

## 组 2：Flag 启停与列表（T3a / T4a）

| # | 场景 | 预期 |
|---|------|------|
| 2.1 | PATCH /status enabled=false 后 evaluate | 恒 false，reason=disabled（联动组 4） |
| 2.2 | 重复停用（已 false 再 false） | 200；不写重复 history（值未变化→跳过 history） |
| 2.3 | 列表筛选 env+enabled+q 组合 | 三条件 AND |
| 2.4 | q 命中 name 或 key（大小写不敏感） | ILIKE 语义 |
| 2.5 | page 超出范围 | 返回空 items，不报错；total 正确 |
| 2.6 | page_size=0 / 101 / 非数字 | 取默认 20 / 截断为 100 / 400 |
| 2.7 | 列表项 rule_count 与实际规则数一致 | COUNT 子查询 |
| 2.8 | 前端 Switch 启停请求失败 | 乐观更新回滚 + 错误 message |

## 组 3：规则 CRUD（T3b）

| # | 场景 | 预期 |
|---|------|------|
| 3.1 | priority 与已有规则撞号 | 409 `DUPLICATE_PRIORITY`，DB `uq_rules_flag_priority` 兜底 |
| 3.2 | PATCH 把规则 priority 改成另一规则现有值 | 409 同上 |
| 3.3 | priority=0 / 负数 / 2.5 / 非整数 | 400 `VALIDATION_ERROR` |
| 3.4 | operator=equals 但 expected_value 是数组 | 400 `VALIDATION_ERROR`（details 指明 expected_value） |
| 3.5 | operator=in 但 expected_value 是标量 | 400 |
| 3.6 | operator=in expected_value=[] 空数组 | 400 |
| 3.7 | expected_value=null / 嵌套对象 / 数组含数组 | 400（D3：只允许标量） |
| 3.8 | expected_value 数字/布尔/字符串标量 | 201 |
| 3.9 | attribute 含前后空格 / 65 字符 | trim 后校验长度；400 |
| 3.10 | 向不存在 flag 加规则 | 404 `FLAG_NOT_FOUND` |
| 3.11 | 删除不存在 / 不属于该 flag 的规则 id | 404 `RULE_NOT_FOUND` |
| 3.12 | 对 disabled flag 增删改规则 | 允许（D4 只影响评估），history 照记 |
| 3.13 | ★ 原子性：构造 history 写入失败，验证规则未落库（或反之） | 事务回滚，两者都不存在 —— API 集成测试用例 |
| 3.14 | rule_updated summary 只记变化字段（如仅 roll_value） | changes 精确 |

## 组 4：评估引擎（T2 单测全覆盖）

| # | 场景 | 预期 |
|---|------|------|
| 4.1 | disabled 且规则会命中 | false / reason=disabled / trace 仅 1 条短路记录（D4） |
| 4.2 | enabled + 无规则 | default_value / reason=default / trace=[] |
| 4.3 | 全部规则跳过（属性全缺失） | default_value / trace 每条 attribute_missing |
| 4.4 | 第 2 条命中，后面还有 3 条 | trace 只含前 2 条（命中即停） |
| 4.5 | 规则乱序传入（priority 3,1,2） | 按 1→2→3 评估 |
| 4.6 | equals：attrs.plan="pro" vs expected "pro" | 命中 |
| 4.7 | equals：attrs.level=42 vs expected 42 | 命中（双侧归一 float64） |
| 4.8 | equals：attrs.vip=true vs expected true | 命中 |
| 4.9 | equals："1" vs 1 / true vs "true" | 不命中，cause=type_mismatch（D3） |
| 4.10 | equals 值不等（"pro" vs "team"） | cause=value_not_equal |
| 4.11 | in：命中列表某元素 / 全不在 | matched / value_not_in_list |
| 4.12 | in 列表混合类型 ["pro", 42]，attrs 42 | 命中（仅同类型比较） |
| 4.13 | attrs 无规则所指属性 | 该规则 skip 继续（D5） |
| 4.14 | attrs 值是嵌套对象，规则匹配其顶层 key | 对象≠标量 → 不命中（in→not_in_list, equals→type_mismatch） |
| 4.15 | default_value=true 全 miss | 返回 true / reason=default |

## 组 5：评估 API（T3b / T4b）

| # | 场景 | 预期 |
|---|------|------|
| 5.1 | flag_key+env 查不到 | 404 `FLAG_NOT_FOUND` |
| 5.2 | body 非法 JSON（少引号/尾逗号） | 400 `INVALID_JSON`，message 含位置 |
| 5.3 | user_attributes 是数组 / 字符串 / 数字 | 400 `INVALID_USER_ATTRIBUTES` |
| 5.4 | user_attributes 含嵌套对象/数组值 | 200（合法），规则只匹配顶层 |
| 5.5 | environment 非法 | 400 |
| 5.6 | 评估结果与直接调 eval 引擎一致（同输入） | 契约一致性 |

## 组 6：批量模拟（T3b / T4b）

| # | 场景 | 预期 |
|---|------|------|
| 6.1 | 5 组 attrs，2 组命中 #2 规则、1 组默认、1 组属性缺失跳过 | summary 计数正确，hit_rules 按次数排序 |
| 6.2 | list 为空数组 | 400 `VALIDATION_ERROR` |
| 6.3 | list > 100 | 400 `BATCH_LIMIT_EXCEEDED` |
| 6.4 | 第 3 组非法（非 object） | 整体 400，details 标注 index=2（预演场景要全对才跑） |
| 6.5 | flag disabled + 批量 | 全部 false + reason=disabled，summary.default_count=0、true_count=0 |
| 6.6 | 前端 textarea 每行一个 JSON，含空行/注释行 | 前端先本地校验：空行忽略、坏行标行号提示 |

## 组 7：操作历史（T3a / T4b）

| # | 场景 | 预期 |
|---|------|------|
| 7.1 | 8 种 action 各触发一次 | summary 结构符合 01-design §8-C schema |
| 7.2 | history 按 created_at DESC 分页 | 最新在前 |
| 7.3 | flag_updated changes 只含实际变化字段 | 见 1.8 / 3.14 |
| 7.4 | rule_deleted summary 记录被删规则的完整摘要 | 删除后仍可追溯 |
| 7.5 | DiffView 渲染 old→new（删除线红/新值绿） | 布尔值渲染为 Tag |

## 组 8：环境复制（T3b stretch / T4b）

| # | 场景 | 预期 |
|---|------|------|
| 8.1 | staging→production 成功 | 201，新 flag enabled=false，规则 priority 全保持 |
| 8.2 | 目标环境已有同 key | 409 `DUPLICATE_FLAG_KEY` |
| 8.3 | target == source | 400 `COPY_TARGET_CONFLICT` |
| 8.4 | 复制后新 flag 的 history | flag_copied，summary 含 source 信息 |
| 8.5 | 复制后原 flag 不受影响 | 源 flag 规则/状态不变 |

## 组 9：前端交互通用（T4a / T4b）

| # | 场景 | 预期 |
|---|------|------|
| 9.1 | 提交按钮重复点击 | loading 期间禁用 |
| 9.2 | 409 错误（key/priority 撞号） | 内联定位到对应表单字段 + 解释文案 |
| 9.3 | JSON 编辑器非法输入 | 不发请求，编辑器下方标错 |
| 9.4 | 各列表/表单空态 | 引导文案而非白屏 |
| 9.5 | 网络断开/后端未起 | 友好错误，可重试 |
| 9.6 | 删除规则二次确认 | Popconfirm |
| 9.7 | key 输入即时正则校验 | 与后端 `^[a-z][a-z0-9._-]{1,63}$` 完全一致 |

## 组 10：复现与工程（T0 / T5）

| # | 场景 | 预期 |
|---|------|------|
| 10.1 | `docker compose down -v && up` + migrate down/up | 干净可逆，无残留对象 |
| 10.2 | seed 重复执行 | 幂等，不产生重复 flag / 重复 history |
| 10.3 | `make dev` 全新 clone 一条命令起全套 | README 步骤可复现 |
| 10.4 | `go test ./...` 全绿 | eval 单测 + API 集成测试 |
| 10.5 | .gitignore 覆盖 .env / node_modules / 构建产物 | 仓库干净 |
| 10.6 | smoke.sh 冒烟序列全过 | 创建→加规则→评估→启停→历史 全链路 |
