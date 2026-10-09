# T0 与波 2 集成交付报告

开始日期：2026-10-09；完成日期：2026-10-10（Asia/Shanghai）。

项目：`C:/Users/77230/FlagForge`；集成分支：`codex/scaffold`。

## 执行范围与环境

用户要求严格执行 T0，随后同时启动 T1、T2、T4a。检测发现依赖缺失后先报告；用户随后明确授权安装全部缺失项。安装及重启后完成 T0 验收，并以 `9a3bf66 chore: T0 scaffold done` 为共同基线，在三个独立 Git worktree 同时调度波 2。

当前环境已可运行：Docker Desktop 4.94.0、Docker Engine/CLI 29.8.2、Compose v5.5.1、WSL 2.7.13.0、Go 1.27.0、Node 24.13.0、npm 11.6.2。项目 Go 指令保持 1.22。migrate CLI 按 v4.17.1 源码安装；源码构建的版本输出为 dev。安装来源和过程、T0 验收详见 [T0-report.md](T0-report.md)。

PostgreSQL 16.15 使用 `flagforge-postgres-1`，宿主端口 55432；服务端 8787、前端 5173。三个 executor 共用已启动的数据库实例，未另起争用端口的 Compose 项目。

## 并行任务与集成提交

| 任务 | 原任务分支及提交 | 集成提交 | 交付 |
| --- | --- | --- | --- |
| T1 | codex/t1-db-models，438788d | 8b87795 | SQL up/down、三表模型、repo、幂等 seed、单测与真实 PG 集成测试 |
| T2 | codex/t2-eval-engine，14bd03b | 27080cb | 纯函数 Evaluate/Normalize、63 个实际用例、trace 与防御行为 |
| T4a | codex/t4a-frontend-base，723f2a3 | 52481e1 | fetch client、MSW、导航、Flag 列表及新建/编辑 |
| T4a 规格修复 | adbe9fd | 1499e26 | 非法 JSON 保留真实错误位置，合法非对象独立校验 |
| T4a 质量修复 | 323c825 | d6aba56 | 分页缩页后实际重新请求有效页；取消响应体读取保持静默 |

主线依次 cherry-pick，各任务源码及报告完整保留。T1/T2 的 go.mod 冲突按依赖并集合并，保留 T0 依赖、Go 1.22 以及双方必要的间接依赖；go.sum 自动合并。未使用 tidy 删除后续任务依赖。

## 独立审查结果与修复

每个任务均接受独立规格审查和代码质量审查，最终均为 PASS。

- T1：DDL 与设计 §4.1 逐字一致；明确 false/空字符串不被 GORM 忽略；事务由后续 service 负责；seed 幂等且整批原子。down 清除全部业务对象，保留共享 pgcrypto 和工具管理的 schema_migrations。
- T2：契约、严格类型和数字归一、稳定优先级、停用短路、命中即停、输入不变及非 nil 空 trace 均满足要求。
- T4a：首轮规格审查发现 INVALID_JSON 丢失位置，修复后 37 项实际 handler 验证通过，独立规格复验通过。质量审查发现末页缩页仍请求无效页、取消 JSON 读取错误 toast 两个 P2；先复现失败再修复，实际组件回调/handler 的 8 项检查、原生 fetch 流式取消及错误路径的 5 项检查通过。最终两类独立复验均通过。

具体测试过程、公共接口与设计解释分别见 [T1-report.md](T1-report.md)、[T2-report.md](T2-report.md)、[T4a-report.md](T4a-report.md)。各任务原始提示词也已归档。

## 主线集成验证

以下检查在合并后的主 checkout 执行并读取退出码；后端检查后仅增加前端修复和交付文档，没有继续修改后端代码。

| 检查 | 结果 |
| --- | --- |
| backend：go build ./... | exit 0 |
| backend：go vet ./... | exit 0 |
| backend：DATABASE_URL=... go test -tags=integration ./... -count=1 | exit 0，repo/seed/eval 均通过 |
| backend：go test ./internal/eval/ -cover | exit 0，100.0% statements |
| frontend：npm run build | exit 0，零 TypeScript 错误，1475 modules，12.04s |
| frontend：npm run lint | exit 0，零错误/警告 |
| scripts/dev.sh 集成启动 | Compose 就绪、迁移 no change、seed 成功，后端与 Vite 正常启动 |
| GET localhost:8787/healthz | HTTP 200，{"status":"ok"} |
| GET localhost:5173 与 /mockServiceWorker.js | HTTP 200 |
| 启动后 PG 查询 | 4 flags / 7 rules / 4 histories，未重复 seed |

### 合并后真实浏览器分页回归

T4a executor 最后一次无法取得浏览器，故其报告诚实注明该场景仅在实际组件代码/请求的内存执行中验证。主线随后成功创建 IAB 临时浏览器，在合并后的前端使用真实 React 页面与 MSW，再补验：

1. 选择「已停用」与「10 条/页」，总数 12，进入第二页，显示功能实验 24、26。
2. 启用功能实验 24 后，总数 11，第二页仍显示功能实验 26，保留有效页码。
3. 启用功能实验 26 后，总数 10，自动回到第一页，显示从账单门户至功能实验 22 共 10 行；上一页/下一页均禁用，筛选与每页数量保持。

实际 switch 数量 10、上一页 disabled；截图已查看且归档为 [Wave2-final-pagination.png](../../frontend/evidence/Wave2-final-pagination.png)。该浏览器验证证明最终可见分页恢复；真实末次请求的 page=1 参数由前述组件/原生 fetch 回归捕获，未声称浏览器工具提供了 Network 面板。

## 工作区与运行状态

三个 executor worktree 清理前均无未提交文件。按 Git blob 校验其全部任务文件与主线一致（go.mod/go.sum 采用已说明的并集合并）。已移除 `.worktrees/t1`、`.worktrees/t2`、`.worktrees/t4a` 临时 checkout，保留原分支和原提交供追溯。直接磁盘 hash 会受 Windows LF/CRLF 转换影响，最终以 Git blob 比较结果为准。

主 checkout 保留可运行开发环境；前端地址 http://localhost:5173，健康检查 http://localhost:8787/healthz。浏览器测试使用内存 mock 数据，刷新恢复演示 fixture。终端 Ctrl+C 停止 dev；`bash scripts/down.sh` 停止数据库且保留卷。

## 当前阶段边界与限制

- T0、波 2 全部完成，无待人工决策。尚未执行波 3 或 T5。
- 后端仍只有 healthz 路由；业务 API、事务历史和评估接口由 T3a/T3b 完成。当前前端默认 MSW；真实后端成功联调待后续任务。
- 详情、规则/历史展示、评估控制台和批量模拟当前为 T4b 占位。
- Vite 生产包约 1.07 MB、gzip 340 KB，存在 chunk 大小提示；构建与 lint 正常通过，未扩大本波次做分包优化。
- AntD 复制按钮已点击，但隐藏浏览器读取剪贴板为空，未声称完整验收操作系统剪贴板；T5 可在可见浏览器检查。
- 未删除数据卷验证完全空环境重建；T1 已验证 up/down/up，完整从零复现属于 T5。
