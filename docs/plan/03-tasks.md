# FlagForge 任务卡 v1.0

> 依赖图与并行编排建议见 00-README.md。每张卡对应 `prompts/` 下同名提示词。
> 预估为 executor agent 实际执行耗时（含自测），供调度参考。

## T0 — 脚手架与环境探测

- **依赖**：无（起点）
- **目标**：探测本机环境（docker/go/node），搭出可运行的空项目骨架
- **交付**：目录结构、docker-compose.yml（PG16）、Makefile、go module、Vite 脚手架、.gitignore、git init + 首次 commit
- **验收**：`docker compose up -d` 后 PG 可连；`go build ./...` 过；`npm run dev` 起来；`make` 目标齐全
- **预估**：0.5h ｜ **风险**：本机缺 Docker/Go → 立即上报，不擅自装大件

## T1 — 数据库迁移与模型层

- **依赖**：T0
- **目标**：migrations（01-design §4.1 原样落地）+ GORM models + repo 基础层 + seed
- **交付**：`migrations/0001_init.{up,down}.sql`、`internal/model/`、`internal/repo/`（含 PG 23505→错误码映射）、`seed/main.go`
- **验收**：migrate up/down 可逆；seed 幂等（跑两次结果一致）；约束名与设计文档一致（错误映射依赖）
- **预估**：1h ｜ **注意**：禁用 GORM AutoMigrate，schema 唯一归属迁移文件

## T2 — 评估引擎与单元测试

- **依赖**：T0（仅需 go module，**不需要 DB**，可与 T1 并行）
- **目标**：`internal/eval` 纯函数实现 + Normalize + 15 用例测试矩阵
- **交付**：`eval.go`、`eval_test.go`、`normalize.go`
- **验收**：`go test ./internal/eval/ -v` 全绿，覆盖 02-edge-cases 组 4 全部 15 条
- **预估**：1h ｜ **注意**：float64 陷阱（提示词有专节）

## T3a — Flag CRUD / 状态 / 历史 API

- **依赖**：T1
- **目标**：flag 创建/列表/详情/编辑/启停 + 历史 API，事务与错误信封基建
- **交付**：`internal/api/`（router、错误信封中间件、handlers）、`internal/service/flag.go`、`internal/dto/`
- **验收**：组 1 + 组 2 + 组 7 的 API 侧行为；API 集成测试（连 compose PG）覆盖 409/404/400/原子性关键用例
- **预估**：1.5h ｜ **注意**：service 层是唯一事务边界；此任务立的错误处理模式会被 T3b 复用

## T3b — 规则 CRUD / 评估 / 批量 / 复制 API

- **依赖**：T3a（复用其错误信封/中间件/repo 模式）+ T2（评估引擎）
- **目标**：规则 CRUD + evaluate + evaluate/batch + copy（stretch）
- **交付**：`service/rule.go`、`service/evaluator.go`（组装：查 flag→短路→查规则→调引擎）、`service/copy.go`
- **验收**：组 3 + 组 5 + 组 6 + 组 8 的 API 侧行为；集成测试补 evaluate 全分支
- **预估**：1.5h ｜ **注意**：expected_value 校验规则（D3）必须前后一致落地

## T4a — 前端骨架 + 列表 + 新建/编辑

- **依赖**：T0（按 01-design 契约 + MSW mock 开发，**不等后端**）
- **目标**：路由/layout/api-client/MSW 基建 + FlagListPage + FlagEditPage
- **交付**：`api/client.ts`、`mocks/handlers.ts`、两个页面 + 主题配置
- **验收**：组 1 / 组 2 / 组 9 前端行为（对 mock 生效）；MSW 关闭后可直接连真实后端
- **预估**：1.5h ｜ **注意**：此任务立的基建被 T4b 复用；409 内联定位到字段

## T4b — 前端详情 / 规则 / 历史 / 控制台

- **依赖**：T4a
- **目标**：FlagDetailPage（规则 Tab + 历史 Tab + DiffView）+ ConsolePage（单次评估 + TraceTable + 批量模拟 + 复制 curl）
- **交付**：`FlagDetailPage/`、`ConsolePage/`、`components/{TraceTable,DiffView,ValueDisplay}`
- **验收**：组 4~组 8 的前端呈现 + 组 9 通用项；关 MSW 连真实后端跑通（若后端未就绪则对 mock 验收）
- **预估**：2h ｜ **注意**：cause 中文文案映射表放前端常量

## T5 — 集成验收 + README + AI 日志归档

- **依赖**：T3b + T4b（全部）
- **目标**：关 mock 全链路真实验收；README 定稿；AI 日志三层归档；冒烟脚本
- **交付**：`README.md`（按 01-design §12 大纲）、`scripts/smoke.sh`、`docs/ai-logs/` 补全、验收清单逐条打勾报告
- **验收**：02-edge-cases **全部 10 组逐条验证**（打勾 + 证据）；`docker compose down -v` 后按 README 从零复现成功；`go test ./...` 全绿
- **预估**：1.5h ｜ **注意**：发现的问题小则当场修、大则记录为已知限制 —— README 诚实披露比隐藏加分

## 总预估

约 10.5h agent 执行时间；三波并行下关键路径约 5.5h（T0→T1→T3a→T3b→T5 中 T5 依赖全部），
加集成调试缓冲，符合题目「建议 6~8 小时投入」的精神（多 agent 并行是效率来源，README 会如实说明工作流）。
