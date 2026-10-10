# 任务 T5：集成验收 + README + AI 日志归档

## 你的角色

你是 FlagForge 的发布工程师：关掉 mock、全链路真实验收、撰写 README、归档 AI 日志、
证明项目在陌生机器可复现。你是质量守门人 —— **发现问题小修、大记录**，诚实披露优于隐藏。

## 前置（全部就绪才开工）

T3b 与 T4b 已完成且各自验收通过（读 `docs/ai-logs/` 全部报告确认）。

## 必读

1. `docs/plan/02-edge-cases.md` —— 你的验收清单（10 组全量逐条验证）
2. `docs/plan/01-design.md` §12 README 大纲、§11 seed 场景
3. `docs/plan/00-README.md` AI 日志三层规范

## 步骤

### 1. 真实集成

- `VITE_USE_MSW=false`（或删 .env.development 里的开关），前端连真实后端
- `docker compose down -v && docker compose up -d` → migrate → seed → 起后端 → 起前端
- 首次联通常见问题自查：vite proxy 指向、错误信封解析、分页参数、日期序列化格式差异

### 2. 全量验收（产出验收报告）

逐条过 `02-edge-cases.md` 组 1~10：每条打勾 ✅/❌/⚠️（部分）+ 一行证据
（curl 输出摘要 / 页面行为描述 / 测试名）。输出到 `docs/ai-logs/T5-acceptance.md`。

- API 组（1/2/3/5/6/7/8 的后端行为）：优先以集成测试跑通为准，curl 抽查关键条目
- 前端组（9 及各组前端行为）：浏览器逐页操作验证
- 组 10（复现与工程）：见步骤 4

### 3. 冒烟脚本 `scripts/smoke.sh`

curl 序列（失败即退出非零）：

```
healthz → 创建 flag → 同 env 重复创建断言 409 → 加规则 → evaluate 命中 →
evaluate 默认 → 停用 → evaluate 断言 false+disabled → history 断言 ≥3 条 →
批量评估断言 summary.total
```

### 4. 陌生机器复现（组 10.3）

- `docker compose down -v`；删除 node_modules 与 go 构建缓存中的项目产物
- 仅按 README（你写完的）从零执行一遍：起 DB → migrate → seed → 后端 → 前端 → smoke.sh 全过
- README 里任何一步需要额外操作而你 README 没写 → 回改 README 再验，直到「照文档必成」

### 5. README.md 撰写（按 01-design §12 大纲，九节齐全）

要点提醒：
- 快速开始给「一条命令」主路径（make dev 或 scripts/dev.sh）+ 分步骤备选 + 本地 PG 备选
- 「规则评估语义」是评分重读区：D1~D5 用人话+三个评估示例讲透（用 seed 数据举例）
- 「设计取舍」精选最有故事的决策（D2 拒绝重复优先级 / D3 严格类型 / D4 短路 / D6 key 不可变 /
  D7 不做删除的 scoping 判断 / D9 last-write-wins 已知限制）
- 「AI 使用说明」：三层日志结构 + 工作流（主线架构 agent → 8 份提示词 → 并行 executor → 归档），
  附 `docs/ai-logs/` 目录树
- 已知限制如实写（未完成项、⚠️ 条目、裁剪掉的 stretch）

### 6. AI 日志归档收口

- `docs/ai-logs/` 补齐：每个任务 `TX-prompt.md`（从 `docs/plan/prompts/` 复制）+ `TX-report.md`
  （缺的找对应 executor 补，或标注缺失原因）
- `session-main.md` 由主线（架构师）提供，你只确认存在并检查目录树与 README 描述一致

### 7. 收尾

- `.gitignore` 终检（.env / node_modules / 构建产物 / pgdata）
- `go test ./...` 与 `go vet ./...` 全绿终验
- 最终 commit + `git tag v1.0.0`

## 验收标准

- [ ] T5-acceptance.md 覆盖 10 组全量条目，❌/⚠️ 都有对应 README 已知限制条目
- [ ] smoke.sh 从零环境跑到全绿
- [ ] README 九节齐全，「评估语义」「设计取舍」「AI 使用说明」三节质量过关
- [ ] docs/ai-logs/ 结构完整（8 任务 × prompt+report + session-main）
- [ ] 干净 clone（或等效重放）按 README 复现成功

## 硬约束

- 你可以修小 bug（ typo、文案、漏掉的空态）；**架构/语义级问题不得擅自改** —— 记录为已知限制并上报
- 不新增功能凑数；可选项（stretch）未完成的如实披露

## 完成后必做

写 `docs/ai-logs/T5-report.md`（模板同 T0 + 验收统计：✅/⚠️/❌ 数量与分布），
然后 `git commit -m "chore(T5): integration acceptance, README and AI-logs archive"` 并打 tag。

## 本轮调度补充（2026-10-10，非原计划正文）

基线8d2524f；独立worktree为C:/Users/77230/FlagForge/.worktrees/t5，分支codex/t5-release。T3a/T3b/T4b已完成独立规格与质量审查后合并。

T5 executor负责README九节、smoke真实curl/Node断言、默认MSW=false、83条证据与日志；root负责真实UI、主线进程、备份后的PG重建、无node_modules/产物的新checkout和独立coldGOCACHE冷复现。先提交可重放prepare版本供root审查合并，再等待实际证据完成最终记录。未经证据不得将pending标为通过。

executor不停止root5173/8787/Docker，不删除数据库卷、现有依赖或全局缓存；自有临时服务使用独立端口与schema，并清理。不要创建下级agents。不擅自改已审查后端/前端业务语义，大问题记录上报。tag v1.0.0由root最终审查合并后在主线创建，避免指向未合并分支。

AI说明保留真实设计/调度摘要与8任务prompt/report，不伪造模型身份、逐字聊天或48小时正式提交。复制与批量均已完成；如实保留admin、last-write-wins、顶层属性、float64、无Flag删除、无前端测试框架、chunk提示和OS剪贴板读回的实际限制。