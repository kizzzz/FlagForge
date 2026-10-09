# T5 执行报告（准备阶段）

日期：2026-10-10（Asia/Shanghai）。worktree：C:/Users/77230/FlagForge/.worktrees/t5；分支 codex/t5-release；基线8d2524f。

## 已交付

- 阅读T5完整提示词、design§11/12、edge全部10组（实际83条）、任务执行报告、原题面audit与三层日志规范。
- README九节：全部核心与四亮点状态、Docker主启动/安装migrate先决条件/本地PG备选、三表与唯一约束、D1–D5和三个seed示例、实际API/curl、测试、取舍、限制、真实AI使用记录。
- scripts/smoke.sh以curl和已有Node≥18断言十步，HTTP/连接/JSON失败非零；唯一测试key，不删除已有数据，README披露保留测试flag副作用。
- .env.development改为false；MSW显式开启命令保留。
- T5-prompt从计划全文复制，session-main原设计摘要保留，未编造逐字聊天或模型身份。

## 实際命令与结果

- 独立端口58025、schema t5_smoke_20261010运行实际cmd/server；DDL从原迁移提取，移除共享扩展行，仅在该schema建表。最初工作目录重复backend导致DDL读取失败，创建了空schema，随即从根目录补执行真实DDL后继续；未影响public。
- 初次smoke按错误的data外壳断言，HTTP201后JSON断言exit1；核对源码后改直接资源对象和reason.type。第二次disabled cause错误断言exit1；核对引擎后改为short_circuited_by_disabled_flag。生产后端没有修改。
- 最终BASE_URL=http://localhost:58025 bash scripts/smoke.sh：十步全部PASS，exit0；测试key smoke-mv19gtbe-5b31c5ec，id5759690a-c71b-4722-bdd7-dc8494177cd2。
- BASE_URL=http://localhost:58025/missing：health返回404，脚本exit1；bash -n scripts/smoke.sh exit0。
- backend go test ./... -count=1：exit0；go vet ./...与go build ./...均exit0。
- 显式DATABASE_URL指向真实55432 PG，go test -tags=integration ./... -count=1：exit0，api9.685s、dto0.260s、eval0.223s、repo0.640s、seed0.494s；不是未配置连接的Skip。
- git diff --check exit0（只有Windows换行提示）；八任务prompt/report归档最终结构待T5收口验证。
- 自有服务Ctrl+C结束；schema DROP后计数0，58025无监听。public只读查询5/9/8（主线并行UI验证），本executor未写public。
- 自动审批曾拒绝Start-Process后台组合（blocked by policy），改受管前台exec会话运行，验证后停止；未停止root5173/8787或Docker。

## 待主线真实证据

本提交为可审查/合并的准备版本，不宣称最终交付完成。root负责真实UI、备份后的PG重建、全新checkout与独立coldGOCACHE、按README冷启动重放。收到证据后逐行完成T5-acceptance全部83条、更新限制与验收统计，再提交最终记录。tag v1.0.0由root最终审查/合并后打，不在本分支提前标记。
