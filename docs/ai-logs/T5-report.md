# T5 执行报告

日期：2026-10-10（Asia/Shanghai）。worktree：C:/Users/77230/FlagForge/.worktrees/t5；分支codex/t5-release；基线8d2524f。准备提交6dac782，版本要求/精确清理修复e3666dc。

状态：T5发布文档与功能验收收口完成，83✅/0⚠️/0❌。准备版本独立规格/质量审查均PASS，最终文档提交交由root复审/合并；版本tag由root在最终合并后创建，本分支不提前打tag。

## 做了什么

- 阅读T5完整提示词、design§11/12、edge全部10组（实际83条）、T0/T1/T2/T3a/T3b/T4a/T4b及wave2执行报告、原题面audit、三层日志规范与主线设计记录。
- README九节：完整核心/四亮点状态、Go/当前前端Node版本、Docker主启动、安装migrate先决步骤、本地PG备选、三表/两唯一约束、D1–D5与三个seed例子、实际API/curl、测试、取舍、限制、真实AI说明。
- scripts/smoke.sh使用curl+已有Node≥18做十步JSON/HTTP断言，失败非零；每次独立key，明确披露保留测试Flag/Rule/History副作用。不新增jq/Python/测试框架。
- .env.development改false；MSW保留显式开发启动说明，最终演示连接Go/PG。
- .gitignore补pgdata/，既有.env、node_modules、dist、backend/bin、日志、.worktrees等覆盖保持；smoke只rm-f自己响应文件+rmdir自己的临时空目录。
- T5-prompt保留计划全文，并明确追加本轮调度补充；8任务prompt/report及session-main均存在。原设计摘要没有改成逐字聊天，也未推断所有模型身份或伪称48h正式提交。
- T5-acceptance保留原边界表全部场景/预期，每条一行实际证据，编号逐项比对83/83无遗漏。

## 关键决策与审查修复

1. **真实契约为准**：核对实际handler/DTO，成功资源直接返回JSON对象，reason是包含type/priority/rule_id/summary的对象；disabled Trace cause为short_circuited_by_disabled_flag。README/curl与smoke采用实际字段。
2. **失败即非零**：HTTP状态、连接失败、非法JSON或语义断言失败均中止。允许重复运行，用时间+随机key避免撞已有数据；产品没有Flag删除入口，脚本不假装无副作用。
3. **当前lock版本要求**：计划/T0的Node≥18是原门槛；package-lock中的部分传递依赖实际要求更高（如eslint-visitor-keys的^20.19.0/^22.13.0/≥24、inquirer的20.17+/22.13+/23.5+、type-fest≥20）。独立规格审查将README Node≥18判为P2，与executor先前检查一致；e3666dc改为推荐且已验证Node24.x（本机24.13.0/npm11.6.2）。准备版本修复后独立规格与质量复验均PASS，root依次合并401af51/748550c。smoke自身仍≥18；未降级依赖或改历史报告。
4. **证据分工**：executor仅操作独立schema/58025；root操作主线UI、进程、备份后的DB重建及新checkout/coldGOCACHE。既有MSW证据明确标来源，不能替代最终Go持久化。

## Executor实际验证

在本worktree：

| 检查 | 结果 |
| --- | --- |
| go test ./... -count=1 | exit0，普通DTO/eval/repo通过，不连接PG |
| go vet ./...；go build ./... | 均exit0，无诊断 |
| 显式DATABASE_URL，go test -tags=integration ./... -count=1 | exit0，api9.685s/dto0.260s/eval0.223s/repo0.640s/seed0.494s；真实PG，无未配置Skip |
| go test ./internal/eval -count=1 -cover | exit0，0.115s，100.0% statements |
| 独立schema的实际cmd/server，BASE_URL=:58025 smoke | 十步PASS，exit0；key smoke-mv19gtbe-5b31c5ec、id5759690a-c71b-4722-bdd7-dc8494177cd2 |
| BASE_URL=:58025/missing smoke | health404，expected200，exit1；不能把错误路由判为通过 |
| bash -n scripts/smoke.sh | exit0 |
| git diff --check | exit0，仅Windows换行提示 |
| prompt源文本完整前缀检查 | PASS；全部8任务prompt/report存在 |
| 83条编号比较/README章节 | 83/83完全相同；9节齐全 |
| git check-ignore --no-index | .env/.env.local/.env.*.local/node_modules/dist/backend/bin/pgdata全部命中；非敏感默认MSW开关文件明确跟踪 |

### 实际失败与修正记录

- 最初DDL读取因backend workdir又写backend/migrations路径失败，仅创建了空schema；立即从根目录补执行原迁移DDL（移除共享扩展行），然后启动服务。未改public。
- 初次smoke错误使用data外壳，真实HTTP201后JSON断言exit1；核对代码改直接对象与reason.type。随后错误disabled cause断言也exit1；改为真实引擎cause后十步全通过。两次红结果证明JSON断言会实际阻断，生产后端未改。
- Start-Process后台组合被自动审批拒绝，原因blocked by policy；改受管前台exec临时服务并在结束时Ctrl+C。
- 清理后58025无监听，t5_smoke_20261010 schema计数0。清理时public只读计数5/9/8（root并行UI操作），executor没有向public写测试数据，没有停止root5173/8787/Docker。

## Root真实Go/PG浏览器证据

默认真实模式，以下均为主线实际浏览器操作，非MSW：

| 场景 | 实际证据 |
| --- | --- |
| seed与评估 | 4seed列表；结账production CN/pro为true/#2、Trace仅2；billing恒false/disabled/一条短路 |
| 批量/JSON/404 | 五组3true/2false/default2/5；空行忽略、坏JSON原始第3行且disabled；重复tokenJSON第3行；格式化可用；404可修正 |
| Flag新建/编辑 | 中文名key t5-ui-persistence，id71f843b0-b321-4ec5-b193-2be344f06428；同key409内联，非法key/required本地错误；编辑key/env禁用，name与defaultfalse→true保存 |
| 规则 | equals null本地拒绝；' plan 'trim；priority409内联；priority2/rollfalse成功；删除前2条、Popconfirm确认后1条 |
| loading防重复 | 仅暂锁自己测试Flag行阻塞真实INSERT，confirm/cancel/attribute isEnabled=false，保存中弹窗保留；T5-real-pending.png |
| 历史Diff | 旧值rgb(207,19,34)+line-through，新rgb(35,120,4)，4boolean Tag；删除后仍完整规则摘要；T5-real-history.png |

root另补齐：停止真实Go后Switch请求500，aria-checked回滚原true；重载列表显示失败/重试，重启Go后点击重试恢复5Flag。搜索never-exists-t5-empty显示新建引导；新Flag初始0规则有添加引导。重启后自有Flag仍为编辑后name/defaulttrue/rules2/history7，再启停后history9，证明Go/PG持久化。development→staging复制201，新iddb418f86-a3da-4011-a258-891a89b35eef停用、priority[1,3]、源不变、flag_copied来源完整；staging→development409UI可恢复。单次defaulttrue说明和skip两行也实际可见。当前tab的MSW日志为空，Go请求日志与PG随机UUID吻合。

root的HTTP补矩阵：PATCH随机不存在UUID404 FLAG_NOT_FOUND；大写/空格/中文/数字开头/65字符五key全部400 INVALID_KEY；prod/空environment全部400 INVALID_ENVIRONMENT。

root在main提交88fcb37保存10张T5-real截图：golden、disabled、default、batch、priority-conflict、pending、history、copy、switch-rollback、empty。Executor已查看golden/batch/copy/rollback/empty/history/pending七张，视觉结果支持对应描述；截图不能证明其不可见的HTTP/PG信息，那些由操作日志/请求/DB证据支持。

## 验收统计与未尽证据

逐条详见T5-acceptance.md；83条功能证据83✅/0⚠️/0❌，包括root新checkout真实dev/smoke与PG重建/迁移/双seed。cold完整Go检查均通过，最后文档提交由root复审后合并。

组分布总数：1=11、2=8、3=14、4=15、5=6、6=6、7=5、8=5、9=7、10=6。各组全部✅。OS剪贴板与npm audit的额外限制单独披露于README§8，不混入83条功能计数。

## 原题面audit闭环与限制

requirements-validation-2026-10-10.md保留其c5ef231早期基线的失败事实。T3a补齐Go生命周期/统一错误/事务历史，T3b补齐规则/evaluate/batch/copy及真实PG失败回滚测试，T4b补齐详情/历史/规则/控制台；T5补默认真实API、smoke、README九节与最终证据。不得把早期失败截图改成当前结果；当前完整Go API、真实前端/PG持久化、事务原子性、全部规则/评估/历史页面、默认真实模式、README九节、冷重建和smoke均以本轮证据关闭对应缺口。完整题面功能验收以83条最终清单为准；原audit的48h正式提交无法验证事实仍保留。

实现边界：固定admin无登录/权限、last-write-wins无乐观锁、顶层属性、JSONfloat64精度、无Flag删除、无前端自动化框架、chunk>500KB提示，均对应README§8。复制与批量均完成，不作为被裁剪stretch。复制curl生成/转义/反馈验证通过，但root首次OS clipboard读回仍空，尚不能证明操作系统剪贴板完整写入；这是额外可选体验验证限制，不计入83条必做清单失败。

48小时接题/正式提交时间不能仅由Git证明，本轮不伪称已对外正式提交作业。session-main为原设计过程摘要，session-completion记录Codex后续调度/集成；不是完整逐字聊天记录。

## Root冷复现证据

- 新checkout分支codex/t5-reproduction，HEAD88fcb37；本轮GOCACHE=backend/bin/cold-go-cache-resume首次Test-Path为false，使用go1.27.0，go mod download + migrate install均exit0。
- npm ci/build/lint全部重跑exit0；294 packages、1497modules、11.19s，chunk1171.05KB/gzip371.03KB；Node24.13.0/npm11.6.2。
- Docker恢复29.8.2；重建前PG6/11/14，latest备份7739bytes，SHA256 639e9d5270e5db61f60b6cbeaf16c74c23097cc47dcb0335fa556e3bd3f8e5ed。
- 实际fresh docker compose down-v/up；migrate up/down/up全部正确，down后business表/索引/序列关系count0。仅保留共享扩展和迁移工具元数据，符合迁移设计。
- 额外npm audit实际报告4项（3moderate/1high），涉及Vite5.4.21 Windows开发服务器路径绕过、esbuild、react-router6；修复建议均涉及major升级，未擅自改已审查依赖栈。README§8披露；不声称audit零问题，不把这项额外依赖检查混进83条功能失败统计。
### 完整主启动与冒烟

- 全新checkout绝对路径C:/Users/77230/FlagForge/.worktrees/reproduction/FlagForge，HEAD88fcb37；新GOCACHE为backend/bin/cold-go-cache-resume，首次不存在。
- up/down/up后双seed均seed complete；实际PG4/7/4，CLEAN_REBUILD_PASS exit0。
- README主命令scripts/dev.sh：Compose Running → migrate no change → seed → build/run Gin:8787 → Vite5.4.21 ready1010ms localhost5173，无需额外未记步骤。
- 真curl health200/statusok、5173 HTML200、5173/api/v1/flags200/total4及真实UUID/UTC，确认代理到Go与PG。
- fresh smoke十步HTTP/JSON全部PASS exit0，key smoke-mv1rlcrt-77c64d86、id853b7291-ded3-42d3-9851-affdec9be17b。
- cold完整Go set-e序列exit0，输出COLD_GO_FULL_PASS；普通test DTO5.765s/eval5.736s/repo1.595s；vet/build无诊断通过；独立eval cover100.0%，1.497s。
- 显式DATABASE_URL的integration -v没有Skip，全部PASS：api14.628s/dto14.098s/eval1.043s/repo11.788s/seed14.326s。覆盖CHECK真实失败的create/update/status/rules/copy回滚及seed整批原子性。
- cold fresh浏览器再次实际点击结账CN/pro：true/#2/两条Trace，评估时间2026-10-10 10:18:30。
- npm audit本轮证据为实际工具输出，没有额外JSON归档文件，不伪称文件存在。
## 最终文件与验收分布

最终新增/更新：README.md、scripts/smoke.sh、frontend/.env.development、.gitignore、docs/ai-logs/T5-prompt.md、T5-report.md、T5-acceptance.md。未改已审查Go/React业务语义、依赖栈或数据库迁移。准备源码已由root合并401af51/748550c；10张真实界面截图已由root在88fcb37归档。

| 组 | ✅ | ⚠️ | ❌ |
| --- | --- | --- | --- |
| 1 Flag创建/编辑 | 11 | 0 | 0 |
| 2 启停/列表 | 8 | 0 | 0 |
| 3 规则CRUD | 14 | 0 | 0 |
| 4 评估引擎 | 15 | 0 | 0 |
| 5 评估API | 6 | 0 | 0 |
| 6 批量 | 6 | 0 | 0 |
| 7 历史 | 5 | 0 | 0 |
| 8 环境复制 | 5 | 0 | 0 |
| 9 前端交互 | 7 | 0 | 0 |
| 10 复现/工程 | 6 | 0 | 0 |
| 总计 | 83 | 0 | 0 |

额外OS剪贴板读回与npm audit依赖风险分别披露，没有遗漏、伪称通过或混入83功能计数。root负责最后停止colddev、恢复其latest备份、复审合并与主线tag；本executor的58025服务/schema已先前清理，未操作root运行资源。
### Root恢复与运行收尾

colddev受管会话81388 Ctrl+C后5173/8787均无监听。root恢复latest public data-only备份，恢复值6flags/14histories/11rules、migration1/dirtyfalse；随后仅删除root自有明确定名t5-ui-persistence、指定ID71f843b0-b321-4ec5-b193-2be344f06428与db418f86-a3da-4011-a258-891a89b35eef的两Flag及级联自有规则/历史。最终真实PG4/7/4，migration version1/dirtyfalse，原4seed UUID全部保留，USER_DATA_RESTORE_PASS exit0。

cold临时服务全部停止；root主checkout重新按scripts/dev.sh启动作最终预览。executor未停止或改动该主线运行资源；smoke在正常使用时仍会保留其新建测试Flag，root此次恢复备份是验收环境收尾，不是脚本自带删除行为。