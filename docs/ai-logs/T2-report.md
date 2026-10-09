# T2 执行报告

执行日期：2026-10-09（Asia/Shanghai）
项目根：`C:/Users/77230/FlagForge`
执行 worktree：`C:/Users/77230/FlagForge/.worktrees/t2`
分支：`codex/t2-eval-engine`；基线：`9a3bf66 chore: T0 scaffold done`
状态：**T2 已完成，全部验收通过，可供 T3b 复用。**

## 环境探测结果

| 检查 | 结果 |
| --- | --- |
| 系统 / Shell | Windows，PowerShell |
| Go | `go1.27.0 windows/amd64`；项目 `go.mod` 保持 `go 1.22` |
| 测试依赖 | testify v1.9.0；只补齐现有版本所需的传递依赖和校验记录 |
| 网络 | 本次下载使用进程级 `GOPROXY=https://goproxy.cn,direct` |
| 数据库 / HTTP / 端口 | 本任务无需访问或启动，未作任何操作 |

## 做了什么

- 阅读 `01-design.md` §5 D1–D5、§7、§8-A / §6.3，以及 `02-edge-cases.md` 组4；完整原始任务提示词归档至 `docs/ai-logs/T2-prompt.md`，包含日期和来源。
- 新增 `backend/internal/eval/eval.go`：Rule / Input / TraceStep / Result 的字段、类型及导出函数签名符合 §7，无额外序列化标签。Evaluate 先处理 disabled；复制规则切片后使用 `sort.SliceStable` 升序评估；仅读取输入，产生已评估规则的 trace，命中即停。
- 新增 `backend/internal/eval/normalize.go`：Normalize 将全部内建有符号 / 无符号整型（含 uintptr）及 float32 转为 float64，其余值保持原样；DeepEqualNormalized 两侧归一后用 reflect.DeepEqual 比较。
- 新增 `backend/internal/eval/eval_test.go`：testify + table-driven，24 个主评估子用例、9 个防御式输入子用例、20 个 Normalize 子用例、9 个 DeepEqualNormalized 子用例及 1 个稳定排序独立用例，共 63 个实际用例。主矩阵完整比较 Result / HitRule / Trace，检查输入未修改和空 trace 的 JSON `[]`。
- `backend/go.mod` 仅增加 `go-spew v1.1.1` / `go-difflib v1.0.0` 两个 testify 必需的 indirect 依赖；`go.sum` 增加上述两者及 testify v1.9.0 的内容校验。未删除或升级 T0 的任何依赖。

## 遇到的问题与解决

1. 第一次运行测试遇到缺失 testify 内容校验；补下载后又发现 go-spew 传递依赖缺少校验。使用 `go get -t flagforge/internal/eval` 补齐必要的两项 indirect 和三项校验，保留全部其他模块。
2. 按 TDD 先写测试，再提供契约类型与无业务行为的占位函数。补齐依赖后 `go test ./internal/eval/ -v` 返回 1，失败来自未实现行为：disabled 的 ReasonType 为空、Trace 为 nil；4.7 数值归一缺失；其他规则与防御式用例均未满足。确认红灯后才实现引擎，随后全绿。
3. 实现后首轮 `go vet`、覆盖率与依赖扫描一次通过；没有修改设计文档或其他任务文件。
4. `git diff --cached --check` 提示归档提示词第54行存在尾随空格；该空格来自原始提示词，为完整保留源文本而保留。代码、模块记录和执行报告无新增空白问题。

## 关键决策

- **§7 与 §6.3：** 保持类型签名，不在引擎 struct 加 JSON tag；短路 TraceStep 的 RuleID 为空，其余字段保持零值。外层 API 负责映射为接口对象。无规则时使用非 nil 空 trace，确保直接序列化为 `[]`。
- **D1 / D2：** 引擎不负责校验重复 priority；遇到相同 priority 时 `sort.SliceStable` 保持输入的相对顺序。输入规则切片先复制，返回命中规则的 struct 也来自该副本，避免修改输入规则本身。
- **D3 / §7：** Normalize 仅归一顶层内建实数类型，不递归转换对象 / 数组，也不解析字符串或 json.Number；它们属于「其余类型原样返回」。严格类型不进行字符串、数字与布尔互转。
- **T2 已知陷阱5 / §6.2：** 规则创建层应拒绝非法 Expected。引擎防御式处理 `in`：Expected 非 `[]any` 时不命中；列表内 null、数组、对象、其他非标量元素跳过；实际属性值非标量时也不命中。全部这类 miss 的 cause 为 `value_not_in_list`，不产生 type_mismatch。
- **§6.2 / §6.3 的未定义非法 operator：** 引擎默认视为不命中，cause 使用已有 `value_not_equal`，继续评估；未新增枚举。合法输入不走此分支，避免将创建校验移入引擎。

## 未尽事项 / 需要人工决策

- **T2 无未完成项，无待人工决策。**
- API 映射、输入校验、数据加载与批量统计属于 T3b，本任务交付纯函数和类型契约。
- 对象 / 数组仅按顶层 key 取值，不支持嵌套路径；非法 `in` 和未知 operator 的防御行为已在上方记录。

## 验收自验清单

- [x] `go test ./internal/eval/ -v`：退出 0，全部 63 个实际用例通过，输出可见组4的全部15个边界编号。
- [x] `go vet ./internal/eval/`：退出 0，无诊断。
- [x] `rg -n 'gorm|gin|pgx' backend/internal/eval/`：退出 1，无匹配（含全部包源码、测试和注释）。
- [x] 4.7 与 4.9 同时存在且通过：真实 JSON float64 对 Go int，反向 int 对 float64 均命中；字符串 `"1"` 对数字 `1`、布尔 true 对字符串 `"true"` 均 type_mismatch。
- [x] `go test ./internal/eval/ -cover`：退出 0，`coverage: 100.0% of statements`，高于 90% 门槛。
- [x] `gofmt` 已应用；Git 变更限于 eval 包、必要模块记录及 T2 提示词 / 报告。

## 测试矩阵对照表

以下用例名位于 `TestEvaluate/` 下；每项均为实际运行 PASS。

| 边界 | 用例名 | 结果 |
| --- | --- | --- |
| 4.1 disabled 短路 | `4.1_disabled_short_circuit` | PASS：false / disabled / HitRule nil / 仅1条短路 trace |
| 4.2 无规则 | `4.2_empty_rules_default_false`、`4.2_nil_rules_default_true` | PASS：default_value / default / HitRule nil / trace JSON 为 [] |
| 4.3 所有属性缺失 | `4.3_all_attributes_missing` | PASS：每条记录 attribute_missing |
| 4.4 第二条命中即停 | `4.4_second_hit_stops_before_three_remaining_rules` | PASS：仅前2条 trace，后三条不出现 |
| 4.5 乱序 priority | `4.5_priority_3_1_2_sorted_without_mutation` | PASS：1→2→3，输入切片不变 |
| 4.6 equals 字符串 | `4.6_equals_string_hit` | PASS：rule_hit / matched |
| 4.7 equals 数字归一 | `4.7_equals_json_float64_vs_int_42`、`4.7_equals_int_vs_float64_42` | PASS：双向归一后命中 |
| 4.8 equals 布尔 | `4.8_equals_bool_hit` | PASS：rule_hit / matched |
| 4.9 严格类型 | `4.9_equals_string_1_vs_number_1`、`4.9_equals_bool_true_vs_string_true` | PASS：type_mismatch |
| 4.10 equals 同类型值不同 | `4.10_equals_same_type_different_value` | PASS：value_not_equal |
| 4.11 in 命中 / miss | `4.11_in_hit_returns_false_roll_value`、`4.11_in_miss`、`4.11_in_single_element_hit` | PASS：matched / value_not_in_list；false roll value 仍为 rule_hit |
| 4.12 in 混合类型 | `4.12_in_mixed_types_numeric_hit`、`4.12_in_mixed_types_strict_string_42_miss` | PASS：仅同类型标量可命中 |
| 4.13 缺失继续 | `4.13_missing_attribute_continues_to_hit`、`4.13_nil_user_attrs` | PASS：attribute_missing 后继续；nil map 无 panic |
| 4.14 嵌套对象属性 | `4.14_nested_object_equals_scalar`、`4.14_nested_object_in_scalar_list`、`4.14_nested_array_in_scalar_list` | PASS：equals type_mismatch，in value_not_in_list |
| 4.15 默认 true | `4.15_default_true_all_miss` | PASS：true / default / HitRule nil |

额外 `TestEvaluateDefensiveInputs` 验证 scalar / typed slice / nil / 空列表 Expected、嵌套对象 / 数组 / null / 非标量列表元素和未知 operator 均不 panic、不命中。`TestEvaluateStablePriority` 验证相同 priority 的稳定顺序，以及返回 HitRule 的 struct 不与输入元素共享地址。Normalize 覆盖所有转换分支；DeepEqualNormalized 覆盖双侧归一与安全结构比较。
