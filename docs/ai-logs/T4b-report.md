# T4b 执行报告

执行日期：2026-10-10（Asia/Shanghai）
worktree：C:/Users/77230/FlagForge/.worktrees/t4b
分支：codex/t4b-detail-console
基线：1769032
状态：T4b 核心与环境复制 stretch 均实现，MSW 实际浏览器验收完成；真实 Go/DB 联调交由 root/T5。

## 做了什么

- 读完 T4b 原始提示、01-design §6.2–6.5/§8/§9/§10/§11、02-edge-cases 组3/5/6/7/8/9、T4a-report；完整提示与调度追加归档 T4b-prompt.md。
- FlagDetailPage 基本信息展示全部 Flag 字段、本地化时间、状态/默认值 Tag、编辑入口；保留原 Shell/路由地址/列表/编辑行为。
- 规则表按 priority 升序、紧凑 JSON、结果 Tag；UI 显式「数字越小越优先（1最高），首条命中即停止；重复优先级会被拒绝」。新建/编辑 Modal 做 attribute trim/长度、priority整数≥1、equals 标量与 in 非空全标量数组即时校验/格式化；409 DUPLICATE_PRIORITY 手动 setFields 定位优先级；删除 Popconfirm。更新后刷新详情与历史。
- HistoryTab 仅首次激活时请求历史，独立加载/失败重试/空态；5条每页加载更多，Timeline覆盖8种 action 图标、操作者与中文相对时间。DiffView 独立复用，changes 逐字段红删除线旧值→绿新值，boolean Tag；rule_created/deleted 摘要，rule_updated 的 rule_id/priority+实际变更，created/copied 对象卡与复制来源。
- Console 单次评估：flag_key/environment/user_attributes JSONobject；本地语法和结构校验、格式化，错误行号；结果大字、三种 reason 中文徽标；独立 TraceTable 全6 cause 中文、高亮命中行、无规则空态、停用短路。404 可修正，400内联，其余错误沿用统一reporter并提供重试。
- 批量每行 object，忽略空行；前端原始行号1起，服务端压缩后的index0起，保存 lines[index] 映射。100上限、true/false/默认比例三卡、按次数降序的规则分布、用户属性/原因明细分页、展开复用TraceTable。
- client.ts 仅追加 RuleInput/TraceStep/Reason/Evaluation/BatchResult 契约类型、ruleApi/evaluationApi/copyFlag；原 request/flagApi/ApiError/取消与reporter约定完全保留。
- 新 detailHandlers/evaluate mock 拆分，复用原 body/error/invalid，新增规则CRUD/409/404/400、单次全部cause/400/404、批量统计/空数组/上限/非法index、复制400/409/201。data.ts 前四组真实对应设计seed：country白名单、plan pro、数字user_id、开发CN返回false、搜索plan/beta_tester、停用billing；26条列表/分页样本保留。
- 核心验收后追加 CopyFlag UI：目标 Select 排除源环境、确认明确新Flag停用、冲突message、成功跳新详情。mock保持原规则优先级、复制独立规则id、source history；flag_copied对象快照避免后续编辑污染历史。
- 未改 README/scripts/backend、依赖栈、默认MSW开关或Compose/数据库。未新增测试框架或持久测试文件；所有新增TSX组件最大55行，已有最大88行，均≤300。

## 关键决策与源码定位

| 决策 | 说明 / 位置（1起） |
| --- | --- |
| 显式 JSON 期望值编辑器 | components/JsonEditor.tsx:3、utils/json.ts:27、FlagDetailPage/RuleModal.tsx:11。避免字符串/数字/布尔隐式推断；parseExpected 与 mock validateRule 使用同一标量边界，JSON null/对象/嵌套数组拒绝。 |
| 规则重复策略 | FlagDetailPage/RulesTab.tsx:20、RuleModal.tsx:26。服务端409后只内联优先级，输入变化清掉旧冲突。 |
| 懒加载与请求取消 | FlagDetailPage/index.tsx:15、HistoryTab.tsx:15。详情GET契约本身带rules，初始规则Tab复用该快照；切换Tab不重拉详情，历史首次激活才GET。详情/历史effect取消旧请求；控制台卸载取消。 |
| 批量行号 | utils/json.ts:34、ConsolePage/BatchEvaluation.tsx:28。空行不计用户组；展示始终使用原始textarea行，后端index通过lines映射。 |
| Trace 中文覆盖 | constants/causeText.ts:2 使用 Record<Cause,string> 编译检查6枚举完整，TraceTable.tsx:4共用。 |
| Diff 与 summary | components/DiffView.tsx:7，不从当前规则猜测过去值；rule_updated契约仅含id/priority/changes，按实际可用数据渲染。 |
| curl 单引号 | utils/json.ts:46，对shell单引号用标准闭合/转义/重开，JSON.stringify保留JSON内容；复制对应最后已展示评估结果的输入快照。 |
| 双Tab可访问性 | SingleEvaluation.tsx:41、BatchEvaluation.tsx:39独立Form name，生成唯一字段DOM id，避免隐藏Tab的同名字段抢label。 |
| Mock 历史故事 | mocks/data.ts:49标注明确为8action展示样本，非真实seed审计记录；业务写入后另记实际变化history。 |

## 实际验证证据

### 临时实际代码/handler验证

全部临时脚本通过 PowerShell stdin → Node/esbuild 内存转译生产源码，无镜像实现、无新增仓库测试文件。MSW Node setupServer 使用实际 handlers，只把相对 /api/v1 地址转成 http://localhost 以供Node fetch；没有启动后端或DB。

- 红阶段：实际handlers检查评估端点缺失，输出 `RED: actual handlers support evaluation: false`、exit1。初次内存import实验因data URL无法解析裸msw失败；修正验证加载器为绝对msw模块URL后获得上述预期红结果。
- 绿阶段第一批 `GREEN actual handlers/parser checks: 57`、exit0：黄金命中/#2/两行轨迹；各cause；billing停用；非法用户属性[]/null/string/number；评估404/非法JSON位置/环境；五组统计3true/2false/2default/hit_counts[2,1]；batch空/101/非法index2与停用批量；priority0/负/小数/string；equals数组/null/对象、in标量/空/null元素/嵌套；规则create201/priority409/edit409/edit200精确changes/delete204/re-delete404；8种history；copy同源400/目标重复409/成功201且规则[1,2,3]；parser原始行号/非object/期望值边界/混合标量/shellQuote。
- 第二批 `GREEN additional actual handler/parser checks: 14`、exit0：无规则default+空trace；development CN返回false；搜索beta_tester第二条命中；顶层嵌套值equals type_mismatch；数字/布尔/字符串规则各创建且严格匹配不同priority；不存在flag加规则404；模拟浏览器Chrome无offset但唯一token报错的真实parseJSON路径定位第3行；101组拒绝与空行映射。
- curl 实际 Git Bash stdin 脚本捕获函数参数，JSON.parse 后与原输入完整比较，exit0。覆盖 O'Reilly 单引号、双引号、反斜线、字面 `$(echo SHOULD_NOT_EXECUTE)`；无命令替换执行。首个验证用Windows bash -c命令行受到Windows/MSYS反斜线再解析影响，改为stdin原样输入后通过，未因此改生产转义函数。

### 实际浏览器（独立5174，MSW）

| 验收 | 实际结果 |
| --- | --- |
| 黄金路径 | production checkout-redesign-2026，country CN/plan pro：true大字，命中#2，trace仅#1不在列表+#2命中，命中行浅绿；T4b-golden.png。 |
| 停用 | billing-portal/development：false、Flag已停用短路徽标，单行「停用短路」；T4b-disabled.png。 |
| 单次JSON | compact object格式化为缩进JSON；多行非法token `}` 第3行立即报错且开始评估disabled；语法错误不进入submit。 |
| 404 | missing-flag显示「Flag不存在：检查key与环境，修改后重新评估」，输入保留、重试按钮可用。 |
| 五组批量 | 3true/2false/default2/5=40%，#2 plan命中2次(40%)、#1country1次(20%)，明细展开2行Trace；T4b-batch.png。 |
| 行号与100上限 | `{}`、空行、`// invalid`标原始第3行且按钮disabled；101行显示101/100及上限错误、按钮disabled。 |
| 规则CRUD | 新建tier null即时报equals标量错误；priority1提交409字段内联；改4后201、规则Tab变4条；编辑roll_value false后结果Tag变化；删除必须确认后恢复3条，历史完整记录。T4b-priority-conflict.png。 |
| 历史 | 5→10→11加载，最终八种action均出现；rule_deleted仍显示#4 tier equals gold→false。读取`.diff-old` computedStyle均rgb(207,19,34)+line-through，新值绿/布尔Tag；T4b-history.png。 |
| 复制stretch | production→development409友好message；取消后选staging，确认明示新flag停用；201跳新id详情，staging/已停用/规则#1~3；历史source=production。T4b-copy.png。 |
| 加载、防重复 | 点击评估/批量时输入与提交disabled且loading；详情/规则/历史独立Spin/Table loading，rule/save使用ref锁避免重复请求。 |
| 可访问性修复 | 首次切换batch观察字段缺少label，定位两个挂载Form生成重复id；加Form name后唯一single-evaluation/batch-evaluation字段id，batch Flag Key textbox可按标签找到。 |

截图均在 frontend/evidence/T4b-*.png，未修改旧证据。

### 工程检查

- `npm ci --registry=https://registry.npmmirror.com`：294 packages安装完成。随后的端口查询未查到监听并使组合命令exit1，安装本身成功；5174以strictPort启动。
- 初次build发现SingleEvaluation一个多余`)`、现有ES2020 lib不支持Object.hasOwn/replaceAll；按错误定位改为hasOwnProperty.call/split.join，无改tsconfig或依赖。随后Fast Refresh lint警告定位ReasonText混合函数导出，改为内部函数。
- 最终 `npm run build`：exit0，零TS错误，1496 modules，built in12.38s；Vite原有>500KB chunk大小提示仍存在（JS约1.169MB / gzip370.40KB），未扩大范围优化分包。
- 最终 `npm run lint`：exit0，零errors/零warnings。
- `git diff --check`：exit0，无空白错误；生成worker仅本机LF/CRLF提醒，没有源码内容diff。
- TSX行数逐文件检查：新增最大BatchEvaluation55，全部≤300。

## 未尽事项与实际限制

- 真实Go API/DB成功联调和83条完整边界回归由root统一执行；本轮只依赖MSW。未伪称事务原子性已用mock验证。
- 网络错误/规则刷新失败/历史失败的重试分支已实现并沿用T4a共享client，不以伪造页面结果声明现场断网验收。root的真实后端错误联调可补该证据。
- 浏览器点击「复制curl」出现成功提示，但隐藏IAB clipboard.readText为空；因此只声明按钮触发和shell命令生成/转义已验证，未声称操作系统剪贴板读回已验证。
- JSON.parse 没有可靠原生位置时，现由独立 JSON grammar 诊断助手定位错误偏移并换算1起行号；重复token/EOF等原限制已在下述P2复审中修复。实际值仍只由JSON.parse产生。
- 删除/新建/复制为浏览器mock临时数据，刷新重置。无新增用户确认或依赖决策。
- 结束时关闭独立tab并停止自有5174 dev；不影响root的5173/8787，端口复核见最终收尾证据。

## 自验清单

- [x] 详情全字段/状态/编辑入口；规则CRUD、409内联、确认删除。
- [x] 历史八种action、summary、Diff和分页加载更多。
- [x] 黄金演示#2 true+两行trace；停用false+短路。
- [x] JSON校验/格式化、404修正路径、批量原始行号与100上限。
- [x] 五组统计、命中分布、展开Trace。
- [x] client扩展与MSW契约全部分支；组件≤300、build/lint。
- [x] 核心后环境复制stretch及独立验收。


## 复审补充：有限数值与收尾

root指出JSON.parse合法指数数字1e400会得到Infinity，JSON.stringify随后变null，可能静默改变用户输入语义。修复前以实际parseExpected(equals/in)、parseObject嵌套属性三场景内存复现，输出 `RED non-finite JSON values accepted: 3`，exit1。

修复 utils/json.ts:3–25：isScalar拒绝NaN/Infinity；hasNonFinite递归检查任意嵌套对象/数组数字；parseJSON完成语法解析后独立拒绝超出有限数值范围，保留用户原始文本。expected编辑器、单次JSON、批量每行、格式化按钮都经同一校验。detailHandlers的用户属性校验亦拒绝非有限嵌套值，原T4a body错误定位逻辑未改。

相同实际函数/handlers绿阶段10项，`GREEN finite JSON actual parser/handler checks: 10`、exit0：equals1e400、in[1e400]、嵌套object/array溢出、batch原始第3行均拒绝；isScalar Infinity/NaN false；1e300与正常嵌套42保留；原始HTTPbody含1e400返回400 INVALID_USER_ATTRIBUTES。现场浏览器输入嵌套1e400显示数值范围内联错误、输入文本保留、格式化与评估按钮disabled；截图T4b-finite-number.png。

最终新代码再次 `npm run build` exit0、1496 modules、12.38s，`npm run lint` exit0零错误/警告，`git diff --check` exit0。临时内存函数/handler共57+14+10=81项断言通过，另有Git Bash curl参数往返验证。

关闭独立IAB tab1，dev session7353已Ctrl+C终止。Get-NetTCPConnection复核5174无监听，root服务仍5173 PID11824、8787 PID33564。未终止root服务。生成mockServiceWorker的工作区标记仅LF/CRLF无内容diff，恢复到HEAD内容，未提交生成源码变更。


## P2 规格审查修复：历史深快照与完整 JSON 行号（2026-10-10）

### 实际红阶段

root审查给出两个具体复现，先以当前生产源码内存esbuild转译 + 真实MSW Node setupServer重现：GET黄金Flag历史id103的rule_created.rule.roll_value为true，PATCH实际规则100…002仅roll_value=false返回200，再GET同历史变为false（updated_at也被改写），完整历史对象比较失败。实际parseObject输入：

```text
{
 "a": {},
 "b": }
}
```

结果为「解析器未提供精确行号」，未包含第3行。输出 `RED historical rule snapshot stable: false ... true false`、`RED failures: 2`，exit1。

### 历史修复与绿阶段

mocks/data.ts:44–47的recordHistory统一使用structuredClone(summary)，在唯一写入入口保存整个summary的深层快照，避免规则对象、嵌套expected数组、changes.old/new、flag对象或copy source的后续变化回写旧历史。未仅在demo rule_created或某单一handler局部shallow clone。

- 实际原GET→PATCH→GET同id历史对象完全不变；直接生产recordHistory对flag_created/flag_updated/rule_created/rule_updated/rule_deleted/flag_copied六action保存嵌套summary后，修改调用者的内层对象/数组/old/new/source/flag仍与原快照完全一致：`GREEN audit snapshot actual handler/data checks: 7`、exit0。
- 再经实际POST/PATCH/DELETE/copy/createFlag handler：rule_created含in数组[US,CA]后改活规则数组；rule_updated后分别改旧/新活数组；rule_deleted后改删除对象数组；flag_copied后改新Flag；flag_created后改新Flag名称，旧历史全部保持写入时内容：`GREEN real CRUD/copy nested audit snapshot checks: 6`、exit0。

### JSON 定位修复与绿阶段

新增utils/jsonSyntaxOffset.ts:4的diagnoseJSONSyntax，只诊断原文、返回首个grammar错误的UTF-16 offset，不创建JSON值。使用显式容器stack（无递归）处理对象key/冒号/值/逗号/end、数组值/逗号/end；扫描字符串转义、4位unicode、非法控制字、数字符号/小数/指数、true/false/null及尾随文本。EOF按原文末尾定位。合法JSON返回undefined，不改变原文。

utils/json.ts:15–24只在JSON.parse抛错且无可靠native offset/line时调用诊断助手，原生可靠位置优先，fallback offset换算1起行。成功值仍JSON.parse，已有hasNonFinite拒绝溢出、equals/in/批量行号规则不变；不再猜重复token出现位置。

- 42个非法fixtures：重复token（含字符串内合法右括号）、多行/嵌套、尾逗号、漏冒号、未引号key/值、EOF、raw控制字、无效escape/Unicode、非法number/literal、尾随文本。每个分别检查直接诊断offset换行、实际parseJSON错误行、强制native无位置异常时的fallback行。
- 16个合法fixtures（空对象数组、标量、数字指数、嵌套、unicode/转义/未配对surrogate）、12000层合法/少一个闭合符的深层JSON，以及finite与确定offset检查，总输出 `GREEN actual JSON diagnostic/parser checks: 147`、exit0。
- 诊断复查发现错误文案回显用户key「position 0」会被旧宽松native regex误认；实际parseObject(`{"position 0":\n }\n}`)红结果第1行，真实错误第2行。改为匹配V8/Firefox原生diagnostic suffix，避免从用户输入片段提取位置。
- 确定seed生成100个合法JSON（string/boolean/number/null/嵌套/控制转义/unicode），各作5次单字符插入/删除，逐一与真正JSON.parse接受/拒绝对照，加snippet回归：`GREEN grammar differential/snippet checks: 601`、exit0。
- 最终source再验10个关键坏JSON的native/fallback双路径、嵌套finite拒绝、batch原始第3行、合法嵌套值：`GREEN final actual parser regression checks: 23`、exit0。

所有验证为临时stdin/内存脚本，直接执行实际源码，不新增框架、依赖或持久测试文件。上述未把旧「无法定位」分支声称为满足规格；它已由grammar fallback替代正常语法错误诊断。

### 最终工程与环境证据

- 最终 `npm run build` exit0，零TS错误，1497 modules，built in10.36s；原有chunk大小提示仍存在（约1.171MB/gzip371KB）。
- 最终 `npm run lint` exit0，零errors/零warnings；`git diff --check` exit0。
- 本次尝试新独立浏览器，cua.createBrowserTab返回「Browser is not available: iab」，cua.getState返回apps=[]/browsers=[]，未重复尝试或伪称现场浏览器结果。此次新增修复证据为实际函数/handler检查；已有7张T4b截图保留。
- 本次仅临时5174 dev session41817已Ctrl+C结束，端口5174无监听；root 5173 PID11824 / 8787 PID33564仍在运行。
- 新增诊断助手约85行，无组件长度、路由布局、client、其他行为改动。
