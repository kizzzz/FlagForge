# 完整题面补齐：主线执行记录

日期：2026-10-10（Asia/Shanghai）。

## 用户授权与起点

用户在完整题面核验后要求「继续完成上述未完成的内容」。受测基线为 `1769032`；当时已完成 T0/T1/T2/T4a，完整题面验证报告明确指出真实 Go 业务接口、详情/规则管理、在线评估、业务历史事务和 README 收口缺失。

继续沿用 `docs/plan/01-design.md` 和任务提示词的已批准设计，不重新改变技术栈或规则语义。此前阶段报告及失败验收证据保留，最终状态由 T5 报告更新。

## 调度与所有权

1. 波 3：T3a 后端与 T4b 前端并行，共同基线 `1769032`。两个 native Git worktree 位于已忽略的 `.worktrees/`，分别使用 `codex/t3a-flag-api` 与 `codex/t4b-detail-console`。
2. T3a 通过独立规格与代码质量审查后合并，并以此为 T3b 基线；T3b 消费 T2 引擎和 T3a 公共错误/事务模式。
3. T4b 通过独立规格与代码质量审查后合并。开发期 mock 不作为最终持久化证明。
4. T3b/T4b 就绪后执行 T5：默认真实 Go API、逐项边界验收、失败即退出的 smoke、README 九节、AI 日志、干净环境复现和最终版本标记。

executor 只改其分配 worktree 内的任务文件；root 负责合并、审查调度、真实全栈验证及最终交付。子任务不创建下级 agents。独立审查先规格后质量，发现问题交还原 executor 修复并复验。

## 当前基线证据与测试纪律

- 创建 worktree 前检查 `.worktrees` 被 Git 忽略；基线 `go test ./... -count=1` 全绿。
- Go API 使用真实 PostgreSQL 集成测试，先验证缺失行为的红阶段，再实现；每测试隔离数据，历史失败注入不得影响演示库。
- 前端遵守设计不新增自动化测试框架；实际 handler/解析与真实浏览器验证，完整链路在 T5 关闭 mock 后再次检查。
- 保留原 5173/8787/55432 运行环境；executor 自有前端端口必须检查占用，不起第二个 Compose 项目。
- T5 清洁复现前备份已命名的本地开发数据库；依赖重装使用新的 checkout，保留用户当前依赖目录与源码。

## 继续执行过程

T3a、T3b、T4b 与 T5 准备版本均经规格审查与代码质量审查后合并。T3a 修复 PATCH 字段存在性、UTC 输出与统一错误；T3b 验证规则/评估/批量/复制和真实 PostgreSQL 失败回滚；T4b 修复 mock 历史引用快照及重复 JSON token 的位置诊断；T5 修正当前锁文件支持的 Node 版本和 smoke 临时文件精确清理。审查失败与复验过程保留在各任务报告。

| 主线提交 | 内容 |
| --- | --- |
| a20dd59 | T3a Flag API、事务历史、实际 PG 测试 |
| 479766d | T3b 规则 CRUD、evaluate、batch、copy |
| fdaa3b1 / 8d2524f | T4b 详情、规则、历史 Diff、评估与修复 |
| 401af51 / 748550c | T5 smoke、真实默认 API、README、复验修复 |
| 88fcb37 | 十张真实 Go/PG 浏览器截图 |

## 主线实际验证结果

- 真实浏览器完成新建/编辑/启停、规则增删改/重复优先级、历史 Diff、首条命中/默认/停用、批量与 JSON 行号、复制冲突、加载禁用、空态和后端停止后的回滚/重试。临时锁只针对 root 自己新建的测试 Flag；服务重启后修改仍持久化。实际请求及截图见 T5 报告，未把 mock 结果当作 PG 证明。
- 调用额度中断后按用户「继续任务」恢复保留状态。丢失退出结果的安装进程不标成功，重新完整执行并检查退出码；不重复已完成的功能开发。
- 新 checkout：`.worktrees/reproduction/FlagForge`，分支 `codex/t5-reproduction`，受测提交 `88fcb37`。初始无 node_modules/backend/bin；恢复测试另用开始不存在的 `backend/bin/cold-go-cache-resume`，未删除全局 Go 缓存或用户主线依赖。
- Go 1.27.0：README 的 mod download 与 migrate 4.17.1 安装 exit0。Node 24.13.0/npm 11.6.2：npm ci、build、lint 全部 exit0；1497 modules，构建 11.19s，chunk 提示 1171.05 KB/gzip371.03 KB。
- Docker 29.8.2 恢复后先核实原数据 6/11/14；public 最新备份 7739 bytes，SHA256 `639e9d5270e5db61f60b6cbeaf16c74c23097cc47dcb0335fa556e3bd3f8e5ed`，保存在已忽略 backend/bin。随后实际 `compose -p flagforge down -v/up -d`，migrate up/down/up；down 后业务表/索引/序列对象数 0。双 seed 后仍 4/7/4。
- 在新 checkout 按 README 主命令 `bash scripts/dev.sh` 启动 Compose/migrate/seed/Go/Vite；health、5173 HTML、Vite 代理列表都为 HTTP200，列表为真实四 seed。十步 smoke 全 PASS，exit0，测试 Flag `smoke-mv1rlcrt-77c64d86` / `853b7291-ded3-42d3-9851-affdec9be17b`。新环境浏览器 CN/pro 为 true/#2，Trace 两行。
- 冷复现完整 Go 命令序列 exit0：普通 test、vet、build、eval cover100.0%、显式 DATABASE_URL integration 无 Skip。integration API14.628s、DTO14.098s、eval1.043s、repo11.788s、seed14.326s；真实 CHECK 失败验证业务与历史原子回滚。
- npm audit 实际报告 4 项（3 moderate / 1 high），来自锁定 Vite/esbuild/react-router 依赖，提示 major 升级；最终报告/README 如实披露，未擅自做架构级依赖升级或声称零告警。
- 冷复现服务停止、端口释放后恢复最新 public 备份；仅清理 root 自建 `t5-ui-persistence` 的两个明确 UUID。原四 seed UUID 保留，最终 4/7/4、migration1/dirty=false。主线重新使用默认真实 API 启动以便交付预览。

最终 T5 逐项验收、独立审查结论与交付版本在 T5-acceptance/T5-report 及后续主线发布记录中列明；未执行远程推送或对外正式提交。
