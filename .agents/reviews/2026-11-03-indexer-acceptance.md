# 内置索引阶段与 schema 证据矩阵（2026-11-03）

基线：`502cfa9078081aa0018dd45af269eabc6c532fdf`；范围：未提交工作树、已有本地验证结果和本次源码审查。没有部署或操作实际配置、数据库、站点、下载器。此文是审查时点快照，后续行动仅在 [R-004](../TODO.md#r-004) 维护。

## 阶段判断

运行时规则管理、统一索引、任务闭环和管理/Home 界面已有实现及本地回归；完整 Cardigann 语义、真实站点证据和默认交付尚未闭合。37 个合成实际解释器场景及 579 个定义静态编译不能替代整个设计的完成条件。

证据分层：源码证明入口和所有者；回归源证明测试意图；终端成功记录只证明实际执行范围。以下“有回归”不表示全部边界已覆盖或真实站点已验证。最近实际 Go/差分结果见[请求与获取复核](2026-11-03-cardigann-requests.md)，Web/持久化历史证据见 R-004；本次矩阵本身不冒充一次新的完整测试运行。

## 设计阶段 A–F

| 阶段 | 当前实现及可核对证据 | 当前不足与结论 |
| --- | --- | --- |
| A 基线与语料 | [固定语料锁](../../scripts/indexer-definitions/corpus.lock.json)、[schema/加载](../../internal/cardigann/loader.go)、[特征工具](../../cmd/moji-indexer-check/main.go)、[真实差分](../../scripts/indexer-diff/README.md)、[来源与预算](../../internal/indexerdefs/source.go) | 下表逐项映射 schema，但全部执行语义尚未认证；[NOTICE](../../internal/cardigann/NOTICE.md)明确发布授权审查未完。初审时 NuGet 仅输出指纹；后续已锁定 36 包/3 个 lock 并实际重建，证据见请求复核。阶段未完成。 |
| B 中性边界 | [indexer 类型](../../internal/indexer/types.go)、[逐站服务](../../internal/indexer/service.go)、[Jackett adapter](../../internal/indexer/jackett.go)、[runtime 来源接入](../../internal/taskruntime/indexer_resources.go)、[来源落盘](../../internal/taskruntime/selected_resource.go)；service/jackett/resources/selected_resource 测试 | runtime/taskflow 生产类型已脱离 jackett.SearchResult；[旧 tracker 接口](../../internal/tracker/tracker.go)仍为 REST 目录桥暴露 jackett.Indexer，不据此声称整个包已删除。新生产寻源边界已接，整体回归仍随最终验收。 |
| C 引擎执行 | 本文下表；实际差分包括 HTML/JSON、结构化查询、POST 登录、文字/图片挑战、torrent/中间下载/magnet | 多处仅有局部样本，不能称完整语义已完成。具体已观察缺口与待判语义见下表。 |
| D 闭环/配置/UI | [真实应用装配测试](../../cmd/moji/indexers_runtime_test.go)的动态安装→内置/Jackett 获取→核心落盘→假 qB；[配置迁移](../../internal/config/layout_test.go)、[规则备份恢复](../../internal/indexerdefs/service_test.go)、[搜索 run](../../internal/torrentsearch/service_test.go)、[GraphQL](../../graphql/moji/types/indexer_management.graphql)、[Home](../../web/src/components/home/HomeIndexers.tsx)及相关 Web/visual 测试 | 本地闭环有证据；未使用真实站点或实际 downloader，尚不能证明真实来源可用性。最终契约生成一致性、浏览器及升级/回退仍须按最终改动统一验收。 |
| E 对照与可靠性 | [差分场景](../../scripts/indexer-diff/cases)、transport/cookies/cooldown/cache/session 测试、[规则作业恢复](../../internal/indexerdefs/service_test.go)、[登录取消](../../internal/indexerlogin/service_test.go)、[诊断](../../internal/indexercheck/service_test.go)、[持久搜索取消/重启](../../internal/torrentsearch/service_test.go)；[引擎基准](../../internal/cardigann/performance_test.go)、[缓存/限流基准](../../internal/indexer/performance_test.go) | 合成对照已执行，不含真实站点、真实 solver/代理；缺完整错误/取消语义差分、相对时钟及剩余 .NET/date/filter 对照。已测合成引擎、搜索缓存、真实共享 gate 的 p50/p95/RSS/请求数，见下方；不能代替真实网络、完整应用和全部资源缓存负载。阶段未完成。 |
| F 默认交付 | 启动不再硬要求 Jackett；[进程生命周期测试](../../cmd/moji/lifecycle_test.go)；[CI](../../.github/workflows/ci.yml)已含全包 Go 与兼容工作流 | 初审时 README/Docker 示例/smoke 仍要求旧 Jackett 配置；本轮已修正并实际通过 amd64 构建/断网 smoke，见下方补充。arm64 本机执行环境缺失，升级回退及完整交付仍未闭合。 |

## v11 schema 逐项定位

依据 [schema_v11.json](../../internal/cardigann/schema_v11.json)，当前 22 个 definitions、120 个具名 properties。所有类型及其具名 properties 在下面列出；动态键与枚举另见后表。这是完整性索引，不是用字段存在代替行为证明。

| Schema 类型与字段 | 所有者/证据 | 审查结论 |
| --- | --- | --- |
| SchemaRoot：id, replaces, name, description, language, type, encoding, followredirect, testlinktorrent, requestDelay, links, legacylinks, certificates, caps, settings, login, search, download | [模型](../../internal/cardigann/definition.go)、[加载/诊断](../../internal/cardigann/loader.go)、[会话](../../internal/indexer/sessions.go)、[TLS](../../internal/indexer/tls_test.go)；编码/次序/语料及动态安装已有回归 | 结构/来源元数据已保留；certificates/requestDelay 接 transport。replaces/legacylinks 未找到运行时迁移消费者，不能宣称自动迁移站点身份/镜像；现有版本 guard 拒绝无效站点，不静默改源。 |
| Caps：categories, categorymappings, modes, allowrawsearch, allowtvsearchimdb | [类别执行](../../internal/cardigann/search.go)、[目录](../../internal/indexeradmin/service.go)、category_search/result 测试 | 类别及模式有回归；后续 row filter 修正已消费 allowtvsearchimdb，并有实际对照。allowrawsearch 的上游宿主资格语义仍需核对，不能把保存字段当成功能完成。 |
| CategoryMapping：id, cat, desc, default | search.go；category_result_test.go、request_order_test.go | ID/描述/默认/顺序已定位和回归；默认类别的上游差异在请求复核明确记录。 |
| Modes：search, tv-search, movie-search, music-search, book-search | search.go、query_episode.go、query_numbers.go、query_derived.go、[命名参数契约](../../internal/indexerquery/parameters.go)、[实际查询场景](../../scripts/indexer-diff/README.md) | 五种路径、15 个 flag 和 IMDb 前缀已有实际样本；命名参数新回归先复现覆盖 typed 字段及小写遗漏，统一 18 字段资格/别名规范化后通过。外部 JSON 适配对未表达的条件拒绝而非忽略，仍不据此宣称全部宿主资格策略同义。 |
| SettingsField：name, label, type, default, options, defaults | [执行默认值](../../internal/cardigann/execution.go)、[管理验证](../../internal/indexeradmin/validation.go)、[站点编辑](../../web/src/components/settings/IndexerSiteEditor.tsx) | text/password/checkbox/select 有实现。checkbox 真值已按实际 Jackett 修正并对照三种默认/覆盖；select 显式默认、覆盖、无默认和显式空值已实际对照，固定库 914 个 select 均有非空默认。defaults 在 Jackett 仅用于 schema 外 multi-select；不据此要求支持该类型。info 展示仍未全部认证。 |
| Login：method, cookies, path, submitpath, form, captcha, inputs, selectors, selectorinputs, getselectorinputs, error, test, headers | [登录执行](../../internal/cardigann/login.go)、login/order 测试、[人工生命周期](../../internal/indexerlogin/service.go) | 五种 method 有分支，form/post/cookie 和 GET selector 等有局部回归；get/oneurl 未进入真实差分。初审发现的 simpleCaptcha 缺少 submitme=X 已修正，合成登录→搜索→取种实际对照通过；失败/取消另有本地回归。 |
| PageTestBlock：path, selector | login.go 的 testLogin/checkSearchAuthentication；Cookie/认证失效测试 | 有执行/局部回归；登录验证与搜索页识别的所有 HTTP/重定向组合尚未完整对照。 |
| CaptchaBlock：type, selector, input | login.go、captcha-text/image/rejected 场景、indexerlogin 测试 | 人工图片/文字已实际对照；可选 solver 属传输执行能力，已有 fake 协议/预算/出口测试，真实服务未验。 |
| ErrorBlock：path, selector, message | parse.go 的 checkErrors、login.go；captcha-text-rejected 场景 | selector 生效。path/message 仅保存但不执行 message 文本；固定 Jackett checkForError 也不读 path。Moji 故意不向公开错误透传站点正文；其余语义须区分上游惯例与脱敏策略。 |
| SelectorBlock：selector, attribute, optional, default, case, remove, text, filters | [字段选择](../../internal/cardigann/selectors.go)、[CSS 扩展](../../internal/cardigann/css.go)、selectors_test.go | HTML/JSON 字段有样本；不能由字段函数样本推断全部 RowsBlock 同名属性都执行，见下行。 |
| Search：path, paths, allowEmptyInputs, inputs, headers, keywordsfilters, error, preprocessingfilters, rows, fields | search.go、parse.go、execution.go、[安装期语法检查](../../internal/cardigann/compile.go) | 请求/执行路径有回归。初审发现 keywordsfilters/preprocessingfilters 未同等预编译，本轮已修正，见下节；全部过滤器参数/执行边界仍不据此判完成。 |
| SearchPathBlock：path, method, followredirect, categories, inputs, inheritinputs, queryseparator, response | search.go、execution.go、request_order/search/category 测试 | GET/POST、继承、类别、raw、次序和分页已对照；followredirect/自定义分隔符等组合的实际语义证据仍不足。 |
| ResponseBlock：type, noResultsMessage | parse.go、response_encoding_test.go、XML/JSON 搜索测试 | JSON/XML 与空结果分支已定位；错误页、noResultsMessage、缺属性/空行等差异仍须行为样本。 |
| RowsBlock：after, dateheaders, selector, attribute, optional, multiple, missingAttributeEqualsNoResults, case, remove, text, filters, count | parse.go、search.go、row_count_test.go、date_headers_test.go、六个 count 与七个分组/日期实际场景 | 已核对固定两家源码：after/dateheaders 仅 HTML/XML 分支，attribute/multiple/count 在 JSON 分支；行级 case/remove/text 未调用通用字段选择器，不能按 schema 继承字段推断必须执行。JSON count/HTML 多执行已修正；日期首个匹配值无效不得借更早日期，查找只回退父元素一次，两处实际差异已修。未完整分组与全部 XML/选择器组合仍未认证，不扩大七个样本结论。 |
| FieldsBlock：动态字段名及 append/noappend/optional 修饰 | 有序 Fields、parseFields/applyField、category_result_test.go、public HTML/JSON 场景 | 声明顺序、自定义 .Result 依赖及类别修饰已回归；其余结果元数据有些只保留 Fields，不等于 Moji 产品全部展示。 |
| DownloadBlock：method, before, selectors, infohash, headers | [获取执行](../../internal/cardigann/download.go)、[magnet 生成](../../internal/cardigann/magnet.go)、download_test.go、download_cancel_test.go、实际下载/magnet 场景 | 中间请求和直接取种有证据；用户已确认沿用 Jackett 公开 tracker，原页/前置页 infohash 两例移回正式 cases 并通过，另覆盖 public/semi-private/private 哈希结果。原始 magnet 不改写，私有哈希不自动公开 fallback。取消原因保留与已确认资源保留有回归；其他 fallback 组合未完整认证。 |
| BeforeBlock：path, pathselector, method, inputs, queryseparator | download.go、download.before GET/POST 空输入测试 | before/原始响应复用有证据；pathselector 全链和其失败/重定向组合证据不足。 |
| InfoHashBlock：hash, title, usebeforeresponse | download.go、TestDownloadInfoHashAndInvalidMagnet、cases/infohash-page 与 infohash-before | 按用户确认的公开 tracker 默认，原页/前置页两条实际对照现已通过。准确 URI 投影另外核对固定 JSON 序列化，详见请求复核；无 pending 目录或隐藏用例。 |
| SelectorField：selector, attribute, usebeforeresponse, filters | DownloadSelector、downloadField；magnet-page/before、download-page/before 场景 | 原页/前置页来源已实际对照，仍须 fallback 与失败分类样本。 |
| RowFilterBlock：name, args | [行过滤器](../../internal/cardigann/row_filters.go)、parseFields；compile.go | 初审发现的 andmatch CharacterLimit/关键词/ID 资格缺口已修正；新增 10 个实际对照场景，明确采用 Jackett 声明式语义，与现代 Prowlarr 的宽松宿主过滤不同。Unicode 大小写和重复过滤器等边界仍未认证。strdump 不输出敏感正文是有意诊断策略。 |
| FilterBlock：name, args | [过滤器](../../internal/cardigann/filters.go)、filters_test.go、query_encoding/response_encoding 测试 | 25 个 schema 名称均有分支，全部 .NET 边界和参数语义尚未认证，见后表。 |
| IndexerCategories：71 个 schema 枚举名 | [标准映射](../../internal/indexercategory/categories.go)、schema、类别回归 | 本轮集合比对 71 个名称全部有 ID；不以集合匹配证明每个数字与上游完全一致。 |

## 过滤器和模板证据边界

| 完整枚举/能力 | 当前证据 | 尚不能据此判定的范围 |
| --- | --- | --- |
| querystring, split, replace, trim, prepend, append | filters.go、filters_test.go；负索引和替换有样本 | 多字符/Unicode 分隔符、空值与越界、全部模板参数组合。 |
| regexp, re_replace | 有预算的 regexp2；lookbehind、捕获组及字符类回归 | .NET 全部语法、超时/取消、替换组边界和不同过滤器列表的安装期验证。 |
| tolower, toupper, diacritics, validfilename | filters.go 的字符串/正规化分支；部分变音符测试 | 文化大小写、.NET UTF-16 与文件名非法字符的逐项语义。 |
| urlencode, urldecode, htmlencode, htmldecode | .NET 旧编码完整 BMP/字节摘要见请求复核；声明编码回归 | HTML 转义细节、URL 非法输入及所有 IANA charset。 |
| dateparse, timeparse, timeago, reltime, fuzzytime | dates.go、固定 Now 的离线样本 | 上游实际相对时钟、日期文化/时区/歧义格式全集。 |
| jsonjoinarray, validate | JSONPath 与 validate 样本 | JSON null/非数组、动态参数及所有上游大小写/顺序语义。 |
| hexdump, strdump | 保留输入的分支，无站点正文日志副作用 | 不表示提供了上游调试日志；这是脱敏诊断策略。 |
| .Config/.Query/.Result/.Categories/.Today，if/else/range、逻辑/比较/join/re_replace | template.go、template_test.go、实际结构化查询/请求场景；编译/步数/字节/深度预算 | 日期默认、查询派生、真假值和枚举/函数组合还未完整上游认证；不暴露任意 Go 方法或脚本执行。 |

## 交付证据的限制

静态语料报告为固定 commit，不等于未来 v11 更新都能执行。规则在运行时下载，schema 和引擎能力仍由 Moji 版本限定；更新必须先检查候选完整库，失败保留活动库。

上述源码审查没有实际配置迁移、真实站点或 solver 验收；不把旧测试源码、CI YAML、既有截图当成当前发布镜像的成功运行。当前仍需完成的原始范围保持在 R-004，不能用这份矩阵替代修复和最终验收。

## 本轮基于矩阵完成的修正

新增 filter_compilation_test 的失败结果证实：关键词/预处理管线的无效正则和参数数量原先可通过 Load，且两类管线未进入静态特征报告。validateSyntax 已将两者与字段 filters 统一检查和计数；实际临时归档 UPDATE 回归证明坏管线不发布、不改变当前可用规则库。

扩大检查后首次全库结果为 578/579，thepiratebay 的关键词过滤使用 .NET IsCJKUnifiedIdeographs，原引擎缺少块别名转换。没有放宽检查或跳过该定义；依据固定 dotnet/runtime v9.0.0 RegexCharClass.cs 的 U+4E00..U+9FFF 边界补齐转换，新增正/反字符类、边界及字面量转义回归。该适配目前仅覆盖实际遇到的 Cyrillic/CJK 两个 BMP block，不能据此声称全部 .NET Unicode/UTF-16 语义已认证。

修正后固定 579/579 静态编译、1 个已有定义警告、0 失败，报告 `/tmp/moji-v11-all-pipelines-verified.json`；query-cjk-keywords 实际 Jackett 对照通过。完整验证结果继续记录在[请求复核](2026-11-03-cardigann-requests.md)，剩余原始范围继续由 R-004 维护。

## 默认交付补充：v3 示例与无 Jackett smoke

[README](../../README.md)、[Docker 文档](../../docs/docker.md)移除必需 Jackett/管理密码假设，新增[索引器操作说明](../../docs/indexers.md)。打包示例改为 v3，默认内置站点为空、Jackett 关闭且无地址/密钥，规则首次显式下载、24 小时检查/手动应用和内置缓存策略明确。moji-config 离线校验成功；新增实际示例的解析、保存副本/回读及源文件不变回归，cmd/moji-config/internal/config 测试通过，旧配置复制迁移测试保留并通过。

实际执行 `DOCKER_DEFAULT_PLATFORM=linux/amd64 make docker-build IMAGE=moji-indexer-acceptance-amd64`，镜像 `dev-502cfa907808-dirty`、`sha256:5a9aa255584ef642e2bf8f52a774c5dd09bc581e246b2d00eaae43f7b71fc9c4`，inspect 确认为 linux/amd64。运行相同 IMAGE 的 docker-test 最终 exit 0：媒体工具、非 root、镜像内 v3 示例、后端/Web/OCI 版本、断网只读启动、无 Jackett/规则库、本地概况与无隐式检查、配置保存、优雅停止、容器替换后的同一 generation/核心/runtime/cache DB 和设置均通过。初次 smoke 到旧固定缓存路径断言失败；改为按 CURRENT 验证真实存储布局，未删除存储断言。日志在 `/tmp/moji-indexer-docker-build-amd64.log` 和 `/tmp/moji-indexer-docker-smoke-amd64-final.log`。

独立 arm64 构建实际尝试后，在第一个 ARM `/bin/sh` 执行时返回 exec format error；Buildx 平台声明不证明本机有可用执行环境。没有生成可验收 arm64 镜像，也没有为了测试注册宿主特权 binfmt。日志 `/tmp/moji-indexer-docker-build-arm64.log`，后续需要可执行 ARM 的隔离环境或既有原生 CI runner，不能把 amd64 成功扩大为多架构完成。版本/发布脚本各 12 项本地测试通过只是脚本逻辑测试，不是远端 CI 或架构发布。

测试只操作本轮命名的镜像、临时容器和卷；未更改正在运行的服务/实际配置/数据库/下载器，未提交、推送或部署。没有界面源码修改，未重复 Web 视觉矩阵；实际镜像构建已完成当前 Web 生产构建。

## 合成引擎性能基线

新增 internal/cardigann/performance_test.go，可显式执行，默认 go test 不运行 benchmark；无耗时阈值或真实 HTTP。使用共享主机 Linux/amd64、Intel Xeon E5-2673 v4、GOMAXPROCS=2，每项固定 100 次。搜索每次新建 Execution，共用已加载不可变 Plan，完整验证 1000 个唯一结果、无 issue/截断和单次 fake 请求；并发项为八个独立执行共享 Plan，每个操作代表整批八次搜索。HTMLFiltered 对标题执行 regexp，对日期执行 dateparse；这些是代表性合成字段，不覆盖每条社区规则的复杂度。

| 负载 | p50 | p95 | 分配 B/op | fake 请求/op |
| --- | --- | --- | --- | --- |
| 小型规则 Load（schema 初始化后） | 0.305 ms | 0.781 ms | 61,792 | 0 |
| HTML 1000 行 | 19.62 ms | 29.83 ms | 8,434,987 | 1 |
| JSON 1000 行 | 18.92 ms | 25.85 ms | 10,013,578 | 1 |
| HTML 1000 行 + regexp/dateparse | 52.44 ms | 67.56 ms | 20,242,062 | 1 |
| 八个并发 JSON 1000 行（整批） | 93.74 ms | 110.84 ms | 80,109,361 | 8 |
| 合作式取消到 worker 退出 | 0.683 μs | 0.803 μs | 5,000（含创建/接收） | 1 |

实测命令先 `GOCACHE=/tmp/moji-go-cache go test -c -o /tmp/moji-cardigann-performance.test ./internal/cardigann`，再以 GOMAXPROCS=2 执行二进制 `-test.run=^$ -test.bench=^BenchmarkCardigann$ -test.benchtime=100x -test.benchmem`。Python 仅启动这一个子进程，以 resource.getrusage(RUSAGE_CHILDREN).ru_maxrss 记录整组进程峰值 39,560 KiB（38.63 MiB），耗时 19.31 s，exit 0；不把每次累计分配量当成驻留内存。报告 `/tmp/moji-indexer-engine-performance.json` 和同名 .log，race 结构验证另见 `/tmp/moji-indexer-engine-performance-race.log`。

取消 percentile 只测 cancel 到协作请求返回/worker 退出，标准 ns/op 还含准备过程；不是网络栈、代理、solver 或最坏正则的退出耗时。当前测量未含缓存字节/命中、共享限流正常与 429 负载、真实网络及完整应用内存；这些仍为 R-004 的性能未完项，不能据此宣称设计性能验收全部完成。

## 搜索缓存与共享限流性能补充

新增 internal/indexer/performance_test.go，以生产 SearchCache 和 HTTP admission/pace/429 gate 测量，HTTP RoundTripper 和公共 DNS 解析均为 fake，没有 socket、真实凭据或外部请求。Linux/amd64/Xeon E5-2673 v4、GOMAXPROCS=2，缓存与 429 各 100 次，正常节流 20 次；每项均验证行数、命中资格、回调/HTTP 次数以及预算或共享冷却截止。仅显式 -bench 运行，没有给普通 suite 增加耗时阈值。

| 负载 | p50 | p95 | 回调/HTTP 次数 | 缓存结果 |
| --- | --- | --- | --- | --- |
| 命中 1000 个候选 | 17.06 ms | 23.71 ms | 每次 0 个上游回调 | 1 条 / 519,871 B |
| 强制刷新 1000 个候选 | 19.02 ms | 30.55 ms | 每次 1 个上游回调 | 1 条 / 519,871 B |
| 100 个不同查询、每次 1000 个候选 | 15.69 ms | 23.42 ms | 每次 1 个上游回调 | 最终 64 条 / 33,271,731 B，低于 33,554,432 B |
| 共享 429 后每批 8 个账号尝试 | 0.104 ms | 0.180 ms | 初始确认 429 一次，此后 100×8 次尝试新增 HTTP 为 0 | 同一来源/出口截止一致，无 admission 槽泄漏 |
| 正常同源连续请求 | 1000.24 ms | 1000.72 ms | 20 次调用恰好 20 次 fake HTTP | 保留生产约 1 秒最小节流，首个请求无等待 |

缓存候选含完整 server-only Resource/Locator，约 520 KB 序列化体；命中仍需反序列化并生成独立候选 ID，所以此基准不把命中解释为零 CPU/零分配，也不能直接与上一节约 55 KB 原始 JSON 比较 CPU 提升。Pressure1000 验证实际驱逐后仍在默认 32 MiB 搜索预算内，不能把该预算扩大成整进程内存上限。

先 `GOCACHE=/tmp/moji-go-cache go test -c -o /tmp/moji-indexer-performance.test ./internal/indexer`，再以 GOMAXPROCS=2 运行 `-test.run=^$ -test.bench=^BenchmarkIndexer(Cache|Shared429)$ -test.benchtime=100x -test.benchmem` 和单独 `-test.bench=^BenchmarkIndexerPacedOrigin$ -test.benchtime=20x`。Python 只启动这两个测试子进程，RUSAGE_CHILDREN 最大子进程 RSS 为 86,064 KiB（84.05 MiB），不相加两进程内存；实际命令、耗时、原始结果见 `/tmp/moji-indexer-cache-network-performance.json`。初次编译修正了测试中误写的 Locator 类型名，正式基准和五项各 1 次 race probe 均 exit 0；相关 vet、文档与 diff 检查通过。

未改生产代码、配置/API/UI，不重复此前已通过的完整 Go/Web/兼容套件；该证据补足搜索缓存和正常/429 gate 测量，不覆盖 torrent ArtifactCache、大型完整应用、TLS/DNS/代理/solver 实际网络或真实站点。后续仍按 R-004 的原始完整验收推进。

## 日期过滤器中间表示与精度

基线 `44906e6`。实际固定 Jackett HTML 对照先复现：带偏移量的 dateparse 返回 UTC 宿主的日期文本（例如 `Wed, 30 Oct 2024 04:41:12 +0`），Moji 原先保留 `+0530`；七位小数秒的合法 .NET 格式原先无法解析、错误地保留输入。源码依据为同 pin 的 [DateTimeUtil](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Utils/DateTimeUtil.cs) 中 ParseDateTimeGoLang/Rfc1123ZPattern，以及 CardigannIndexer 的日期过滤和 date 字段投影。

修正集中于日期实现：支持 1–7 位 f/F 小数秒；过滤后的中间值与 `.Result.date` 统一为 UTC 和单 z 小时偏移文本；最终日期解析识别该中间值并保留时刻，不把 `+0` 当成未知日期。非法格式继续保留原文。新增纯回归核对正/负半小时时区、跨日时刻、必需/可选小数位和二次解析；新增五条真实解释器场景核对 HTML 日期过滤、正/负偏移、七位小数、内部空白失败控制、`.Result.date` 及最终发布时间。

完整实际对照 92/92（88 正常、4 精确预期拒绝），0 未解释/5 原有已解释差异，报告 `/tmp/moji-indexer-diff-dates-final.json`；未改参考程序或比较器。完整 Go、cardigann/indexer 完整 race 与相关 vet 通过。初次测试变量缺少显式 Scalar 转换导致编译失败，修正后完成全部检查。没有 API/UI/Load/schema 变化，不重复 Web 或静态全库编译；该证据覆盖明确日期格式，不代表全部文化/相对时钟语义已认证。

初始 JSON 探针还识别出独立宿主行为：JSON.NET 会将 ISO 日期字符串读成日期类型，ToString 返回如 `10/30/2024 10:11:12`，而 Moji 当前保留原始 ISO 字符串。HTML 场景已隔离此因素，没有通过忽略差异获得通过；其 JSON 语义仍需单独修复/验收。旧 Go 风格日期 layout 被固定 v11 schema 明确拒绝，未放宽 schema 或将它计入合法 v11 通过样本。UTC 宿主对照不等于所有部署时区均已认证；真实日期、相对时钟和 JSON 类型转换边界继续由 R-004 跟踪。

## 附件缓存负载与取消测量

基线 `8c8bd52`，新增显式 `internal/indexer/artifact_performance_test.go`；不改生产缓存、API 或配置。Linux/amd64、Xeon E5-2673 v4、GOMAXPROCS=2，每项 100 次；输入是约 1 MiB 的合法合成 torrent，获取回调只提供内存字节，不访问 HTTP。普通测试已有预算/取消/迟到成功/关闭回归，本轮相关测试通过。

| 负载 | p50 | p95 | 获取回调/op | 最终缓存 |
| --- | --- | --- | --- | --- |
| 已热命中 | 0.431 ms | 1.616 ms | 0 | 1 条 / 1,048,803 B |
| 每次清空后的冷获取 | 6.805 ms | 14.111 ms | 1 | 1 条 / 1,048,803 B |
| 100 个不同原始附件（同 infohash、不同正文） | 5.946 ms | 8.144 ms | 1 | 63 条 / 66,074,589 B，低于 64 MiB |
| 每批 8 个同资源调用方 | 13.784 ms | 18.019 ms | 1 | 100 个 locator 共用 1 个正文 / 1,048,803 B |
| 64 个已准入获取任务取消到调用方及 worker 全退出 | 0.215 ms | 0.321 ms | 0（仅测取消） | 无 worker/等待者/失败缓存残留 |

命中仍复制不可变正文；并发共享项包括刻意等待八个调用方加入、校验及独立复制的开销，检查各自 opaque ID 和完整字节。压力项使用相同 infohash 但不同原始正文，证明没有用 hash 将不同 torrent（可能包含不同 tracker）错误合并。四项获取基准单一测试进程峰值 RSS 150,540 KiB（147.01 MiB），不将 64 MiB 缓存预算当成进程上限；取消基准单独运行，计时排除准入准备。

命令为先 `go test -c`，再显式 `-test.run=^$ -test.bench=^BenchmarkArtifactCache$ -test.benchtime=100x -test.benchmem`，取消项同样执行 `^BenchmarkArtifactCancellation64$`；`GOCACHE=/tmp/moji-go-cache`。报告 `/tmp/moji-artifact-performance.json`、`/tmp/moji-artifact-cancellation-performance.log`；五项各 1 次 race probe 均通过。该测量证明合成附件处理与预算，不代表网络延迟、真实站点、完整任务或任意规模的内存保证。

## 内置来源完整任务队列负载

新增 `cmd/moji/indexers_builtin_pipeline_test.go`，补足此前应用级安装/任务测试实际主要经过外部 Jackett 适配器的证据缺口。使用正式 `newIndexerRuntime`、临时 Manager 三库、运行时归档安装、网络配置及站点管理，关闭 Jackett；通过 taskflow 并发持久接收八个不同番号的任务，结束请求 context 后才启动正式双 worker。断言接收阶段零网络/零下载副作用。

执行链为：持久队列 → 内置 Cardigann 解释器 → 正式受控 HTTP 与会话/节流 → 每站两条候选 → 获取 sample 和正式 torrent → 已有文件名 EXCLUDE 规则排除 sample → 将正确原始字节及内置来源保存到核心库 → 模拟 qB。模拟下载器在调用发生时重新读核心库，核对来源 kind/实例/规则修订及原始字节与提交数据一致；八项均只提交一次，每项搜索/两个附件各一次。规则只下载一次，不访问 Jackett。

网络目标采用公开字面 IP，连接仅指向显式配置的本地自答 HTTP 代理；代理校验绝对请求目标，直接生成合成响应、不转发、不接触真实站点或外部 DNS。保持生产约一秒同源节流，没有测试专用放行或改变网络预算。

独立测试二进制以 GOMAXPROCS=2 执行，八次接收到进入 DOWNLOADING 的观测 p50 12.135 s/p95 23.128 s，共 24 个 HTTP 和 8 次模拟 qB，峰值 RSS 37,540 KiB（36.66 MiB）。时间包含队列等待及最多约 100 ms 的状态观察间隔；只有八个样本，不能推算真实吞吐或可靠率。报告 `/tmp/moji-builtin-pipeline-measured.json`。正式测试、同一完整链路 `-race` 和相关包 vet 通过；初次测试误用来源字段名 `RuleRevision`，改为实际 `BundleRevision` 后编译及后续运行均通过。没有生产代码/API/UI 变更，本轮不重复无关 Web/差分矩阵；真实 qB/站点、TLS、真实代理/solver 和程序升级回退不由此证明。

<a id="incremental-delivery"></a>
## 按用户要求分批提交

用户要求先修复已知问题、将已有更改逐步提交，再继续验收。本轮从 `502cfa9` 开始拆分；以下均为本地提交，没有推送、部署或操作实际配置/数据库/下载器。

| 提交 | 范围 |
| --- | --- |
| `9f4a05a` | 查询、类别与种子解析基础 |
| `899c398` | Cardigann v11 引擎与固定编码映射 |
| `35d3560` | 存储迁移与索引数据持久化 |
| `f456f4a` | v3 配置、离线配置工具及示例 |
| `73ea81f` | 运行时规则库与版本生命周期 |
| `ac0d498` | 可选 solver 协议客户端 |
| `40f4d88` | 中性索引服务、网络/会话/缓存与 Jackett 适配 |
| `9a22f2c` | 隔离诊断与登录挑战 |
| `fe5fe39` | 站点、网络与策略管理 |
| `37c25a0` | 持久搜索与显式准备下载 |
| `74d2ac7` | 应用组装、任务链路与 GraphQL/后端生成绑定 |
| `b95ebcd` | Web 管理、Home、搜索与前端绑定 |
| `36ed357` | 浏览器行为、合成夹具与视觉基线 |
| `5cb998c` | 固定语料与实际 Jackett 差分工具 |
| `baa734c` | CI 兼容门与无 Jackett Docker smoke |

基础和领域批次分别在仅含当前暂存内容的导出树验证，后端集成批次完整 Go 回归通过。Web 76 文件/428 项、i18n 10 项及生产构建通过；固定 Playwright v1.61.1 容器相关 206 项全部通过，未更新基线，报告没有失败重试。CLI、比较器 17 项、依赖锁检查 5 项、版本/发布脚本各 12 项与 smoke shell 语法检查通过。PNG 合成素材明确标记为二进制，避免 Git 将其文件头误判为文本空白。

交付文档另作一批，完成后先验收当前提交的新 amd64 镜像。前述旧镜像只是历史时点证据；本轮浏览器通过不代表真实站点、ARM64 或远端 CI 已通过。活动未完项仍只在 R-004 维护。

### 已提交源码的新镜像验收

交付文档提交 `0a34d43` 后工作区干净，实际执行 amd64 的 `make docker-build` 和同 IMAGE 的 `make docker-test`，均 exit 0。镜像版本 `dev-0a34d4383cfe`，ID `sha256:38fdecb3e05ff58ad2382e4f84c44d47e3d2925728c0a9fd49906f9a088b2111`；inspect 确认为 linux/amd64，OCI revision 为 `0a34d4383cfe1c42b0743af2547b93615cf56486`。

本次验证当前镜像内的 Web/后端版本、非 root 与媒体工具、无网络/只读根启动、无 Jackett/规则库的本地概况、没有隐式规则检查、配置保存、优雅退出和容器替换后的同一 generation/三库及设置保留。仅使用独立临时容器/卷，无生产服务操作。日志 `/tmp/moji-indexer-committed-amd64-build.log` 和 `/tmp/moji-indexer-committed-amd64-smoke.log`；ARM64、真实站点/代理/solver 与远端 Actions 未由此验证。

### select 空值边界修正

继续验收时，固定实际 Jackett 的两条红报告均证实：Moji 在没有默认值或用户明确配置空字符串时，错误地选择排序后的第一个选项并加入请求。删除引擎与管理投影的这一回退；保留非空未知选项拒绝。编辑器显示“未选择”，不把浏览器原生第一个选项误表示为当前值；清除已配置 select 后显示规则声明默认值，提交仍仅发送 clear。

新增两条正式合成场景，四种 select 默认/覆盖均无例外通过。完整差分现为 87/87（83 正常、4 精确预期拒绝），0 未解释/5 原有已解释差异，报告 `/tmp/moji-indexer-diff-select-final.json`；未改参考 runner 或比较容差。完整 Go、相关 race/vet、Web 编辑器 7 项、i18n 10 项及生产构建通过；固定 Playwright 容器 indexers.spec.ts 36 项无更新/无重试通过，桌面/手机空值截图人工核对、无横向溢出或浏览器/GraphQL 错误。此修正发生在上述 `0a34d43` 镜像之后，该镜像不包含本修正；没有操作实际配置或站点。
