<!-- 提示词归档日期：2026-10-09（Asia/Shanghai）；来源：docs/plan/prompts/T2-eval-engine.md -->

# 任务 T2：评估引擎与单元测试

## 你的角色

你是 FlagForge 的核心算法工程师。`internal/eval` 是全项目的评分重心：纯函数规则评估引擎。
它不碰数据库、不碰 HTTP —— 这份纯粹性就是可测性的来源。测试矩阵必须全部覆盖。

## 必读（动手前读完）

1. `docs/plan/01-design.md` §5 决策 D1~D5、§7 评估引擎规格（**类型签名逐字实现**）、§8-A Trace 语义
2. `docs/plan/02-edge-cases.md` 组 4（15 条 = 你的测试矩阵）

## 交付物

`backend/internal/eval/` 三个文件：

### 1. `eval.go`

按 §7 的类型签名实现 `Evaluate(in Input) Result`：

```
语义（按序）：
1. !Enabled → 返回 false / reason=disabled / trace 仅一条
   {RuleID:"", Cause:"short_circuited_by_disabled_flag", Matched:false}
2. 规则按 priority 升序评估（传入乱序也要正确排序；sort.SliceStable）
3. 每条规则：
   a. 属性不存在于 UserAttrs → trace 记 attribute_missing，continue
   b. equals：Normalize(属性值) 与 Normalize(Expected) 比较
      - 类型不同（string/number/bool 互斥）→ type_mismatch
      - 同类型不等 → value_not_equal
      - 相等 → 命中，返回 RollValue / reason=rule_hit / HitRule
   c. in：Expected 是 []any，任一元素与属性值严格相等（同类型）→ 命中；否则 value_not_in_list
4. 全部未命中 → 返回 DefaultValue / reason=default / HitRule=nil
```

### 2. `normalize.go`

`Normalize(v any) any` —— 导出函数，service 层（T3b）会用它归一 DB 侧 jsonb 值：

- json.Unmarshal 到 `any` 的值已经是规范形态（数字=float64）；Normalize 的职责是
  **兜底归一**：int/int64/float32/uint… → float64；其余类型原样返回
- 附带 `DeepEqualNormalized(a, b any) bool`：两侧先 Normalize 再 reflect.DeepEqual

### 3. `eval_test.go`

table-driven（testify），**覆盖 02-edge-cases 组 4 的全部 15 条**，每条用例名引用编号
（如 `TestEvaluate/4.9_equals类型不匹配_字符串1_vs_数字1`）。禁止仅靠 happy path 测试。

## 本任务已知陷阱（AI 生成代码最易翻车处，逐条自查）

1. **float64 陷阱**：`encoding/json` 把所有 JSON 数字解码为 `float64`。
   属性 `42` 与期望 `42` 若一侧经 json.Unmarshal、另一侧是字面量 `42`(int)，直接 `==` 必挂 —— 
   一切比较必须走 `DeepEqualNormalized`
2. `UserAttrs` 为 nil / 空 map：等同于所有属性缺失，正常走 default，**不得 panic**
3. `Rules` 为 nil/空：直接返回 default（组 4.2）
4. `HitRule` 返回指针：命中时指向命中的规则，未命中/短路时为 nil —— 调用方（T3b）依赖 nil 判断
5. in 的 Expected 元素含嵌套结构：设计上创建时已被 T3b 拒绝，但引擎遇到时**不得 panic**，
   按不命中处理（防御式，注释说明）
6. 禁止 `import` 任何 GORM/Gin/PG 相关包 —— 验收会 grep 检查

## 验收标准

- [ ] `go test ./internal/eval/ -v` 全绿，输出可见 15+ 用例名
- [ ] `go vet ./internal/eval/` 通过
- [ ] `grep -rE 'gorm|gin|pgx' backend/internal/eval/` 无结果
- [ ] 用例 4.9（"1" vs 1）与 4.7（42 vs 42）同时存在且都通过 —— 类型严格与数值归一的双重证明
- [ ] 覆盖率：`go test ./internal/eval/ -cover` ≥ 90%

## 硬约束

- 类型签名、cause 枚举、reason 枚举与 01-design §7 逐字一致（T3b 按此契约组装 API）
- 不读数据库、不起 HTTP；发现设计规格有歧义 → 按你认为合理的解释实现，并在报告
  「关键决策」里写明条款与理由

## 完成后必做

写 `docs/ai-logs/T2-report.md`（模板同 T0，额外附：测试矩阵对照表——15 条边界 → 用例名 → 结果），
然后 `git commit -m "feat(T2): pure rule evaluation engine with full test matrix"`。
