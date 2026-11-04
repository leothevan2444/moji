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
| querystring, split, replace, trim, prepend, append | filters.go、string_filters.go 与回归；split/trim 首 UTF-16 单元及 querystring 大小写/重复键/容错解码、HTML 属性、模板 replace 已实际对照 | 全部非 ASCII 键和模板组合；会产生无配对代理项的 UTF-16 输出明确拒绝，不能算完整无效文本互操作。 |
| regexp, re_replace | 有预算的 regexp2；lookbehind、捕获组及字符类回归 | .NET 全部语法、超时/取消、替换组边界和不同过滤器列表的安装期验证。 |
| tolower, toupper, diacritics, validfilename | filters.go 的字符串/正规化分支；不变文化土耳其 I/希腊/补充平面与 Linux 文件名字符集实际对照；部分变音符测试 | 其他文化、Unicode 版本/UTF-16 及 Windows 实际运行；Moji 文件路径安全验证不归此规则过滤器。 |
| urlencode, urldecode, htmlencode, htmldecode | .NET 旧编码完整 BMP/字节摘要见请求复核；声明编码回归；HTML 引号/Latin-1/BMP/补充平面 entity 实际对照与往返 | URL 非法输入、所有 IANA charset 与全部 HTML 解码宽松规则。 |
| dateparse, timeparse, timeago, reltime, fuzzytime | dates.go/relative_dates.go/fuzzy_dates.go；绝对日期与 Unix 秒完整场景；两组间隔 38 输入及两组 fuzzy 日历 44 输入实际原函数固定时钟对照；date/dateheaders 共用日历解析 | DateTimeRoutines 的其余自由文本/仅时间默认日期推断、实际宿主时区/DST、其他文化及歧义格式；Jackett 外层未来发布时间钳制尚待核对。辅助函数对照不计为完整搜索场景。 |
| jsonjoinarray, validate | 单 token 子节点枚举、null/标量空结果、对象声明顺序、嵌套 JSON/日期与大整数实际对照；validate 固定分隔符/顺序去重实际对照 | 所有 JSONPath 方言、动态参数组合及 JSON.NET 非标准 JSON 输入；不将样本扩大为全部类型兼容。 |
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

## JSON 日期标量取值

基线 `631932c`，继续处理上节识别的差异。固定实际 Jackett 红报告确认 JSON.NET 在读取 JSON 时识别 ISO/Microsoft 日期，Cardigann 的 Value<string>()/数组拼接输出固定日期文本，之后才执行 case/filters。Moji 原先一直暴露原始字符串。另核对 JSON.NET 13.0.3 的 [DateTimeParser](https://github.com/JamesNK/Newtonsoft.Json/blob/13.0.3/Src/Newtonsoft.Json/Utilities/DateTimeParser.cs) 和 [DateTimeUtils](https://github.com/JamesNK/Newtonsoft.Json/blob/13.0.3/Src/Newtonsoft.Json/Utilities/DateTimeUtils.cs)，用 Go time/regexp 独立实现有界日期文本投影，未新增依赖或移植其源码。

`jsonScalarText` 统一处理合法 ISO/微软日期，UTC 宿主文本格式为 MM/dd/yyyy HH:mm:ss；显式偏移先换算时刻，微软毫秒时间不重复套用后缀偏移。支持七位小数、合法午夜跨日和日期范围端点；纯日期、非大写 T、无效日历、八位小数等维持原文。字面 Text 与 HTML 不走这一 JSON 隐式投影；普通数值/布尔/null 处理不变。投影位于取值边界，不替换整个 JSON 树或声称提供 JSON.NET 的全部动态类型/比较器。

新增五个实际场景：23 种原始日期/非日期值、数组拼接、case、jsonjoinarray、JSON 取值→日期 filter→date 字段；日期过滤器面对已转换文本的失败原文回退与最终发布时间一致。新增本地回归覆盖边界、字面 Text 不变及过滤器执行次序。完整实际差分 97/97（93 正常、4 精确预期拒绝），0 未解释/5 原有解释，报告 `/tmp/moji-indexer-diff-json-dates-final.json`；无新增比较例外或参考程序修改。完整 Go、cardigann/indexer 完整 race 与相关 vet 通过。仅引擎与测试/证据变更，不重复未受影响的 Web、静态全库和镜像；来源/许可、其他过滤器和动态日期比较、真实时区/站点等剩余范围仍归 R-004。

## 字符串过滤器的单字符参数与 HTML entity

基线 `2821cb2`。固定语料 `5f5db4e` 的归档摘要再次核对为 `b266646ab2662c260ac6b0fad5f99794b200784f3ac865b521820e8e6814a472`；通过生产 Load 对全部 579 个定义进行临时参数遍历，统计 split 40 处、trim 32 处，没有多字符/空参数/补充平面参数，htmlencode 为 0。报告 `/tmp/moji-trim-split-corpus.json`；临时检查程序已删除，未将生产 YAML 纳入代码。这只说明固定语料用法，不限制未来合法 v11 定义。

实际 Jackett 对照确认普通 BMP 多字符参数原实现已经正确取第一个字符，没有为此改坏现有行为；红报告另复现 split/trim 将补充平面 rune 当成 C# char 的差异、显式空 trim 被忽略，以及 HTML 编码实体拼写不同。新 string_filters.go 保留 BMP 快路径，补充平面按首个 UTF-16 代码单元处理；空 trim 产生字段错误，与上游可选 description 取值失败后的空字段结果一致。HtmlEncode 使用上游的双引号命名实体、Latin-1 和补充平面数字实体，保留普通 BMP 文本；原先 Go EscapeString 的双引号数字实体及非 ASCII 原样输出已修正。

新增六条正式场景（HTML 编码包含五组原始值），固定实际对照 103/103（99 正常、4 精确预期拒绝），0 未解释/5 原有解释，无新比较例外；报告 `/tmp/moji-indexer-diff-string-filters-final.json`。本地回归覆盖负索引、中日韩/补充平面参数、空 trim、无配对输出拒绝与 HTML entity 往返；完整 Go、cardigann/indexer 完整 race 和相关 vet 通过。没有 API/UI/schema/Load 修改，不重复 Web、镜像或全库编译。

无效 UTF-16 边界明确保留：参考程序对 `trim("😀VALUE😀", "😀")` 的中间值产生无配对代理项，实际写出失败 EncoderFallbackException；未修改参考序列化或将其算作通过样本。Moji 在这一边界返回字段错误，不默默替换或丢字符；有效 UTF-8 输出的补充平面 split/trim 已实际对照，但不能据此声称全部无效 UTF-16 管线互操作。初次临时语料检查的绝对路径拼接有误，修正后完成全部 579 定义遍历；不是规则加载失败。所有测试仅用合成数据/固定归档，没有真实站点或下载器操作。

## 对照回放并发隔离（2026-11-04）

执行过滤器回归时发现工具缺陷：两次 compare.py 若共用 reference_root，会覆盖同名 scenario/result 文件。实际同时运行整组和 HTML 专项曾让 tested-download/HTML 场景读到其他场景的结果，旧 `/tmp/moji-indexer-diff-filter-tail-final.json` 与 `/tmp/moji-filter-tail-probes/html-control.json` 因此不作为可靠验收证据；没有把这一串场当成引擎失败或放宽差异匹配。

新增可重复并发回归，以 barrier 保证两个调用都进入参考执行，旧代码确实仅使用一个 scenario 路径而失败。改为每次在参考根下分配独立临时目录，Moji/参考进程只读取该目录，结束或异常时清理；固定二进制仍可共享，不修改 runner 源码或要求重建参考。18 项比较器与 5 项参考锁检查通过。实际并发复核相同参考根的 tested-download 与 filter-querystring-html 均通过，0 未解释/1 原有已解释差异，报告 `/tmp/moji-indexer-diff-concurrent-control.json`；整组重新运行的结果另行记录。

## 查询参数与剩余纯字符串过滤器（2026-11-04）

基线 `dd79c20`，工具隔离修复单独提交为 `e42267c`。实际固定 Jackett 红报告复现五类差异：querystring 键应不区分大小写并取首值，不应重复 HTML 解码，允许分号及原样保留无效百分号/UTF-8；validate 仅按声明的固定字符集分词、保留参数顺序并以不带空格的逗号输出；validfilename 应按宿主原生字符集而非固定 Windows 集合；不变文化的土耳其 I 大小写应保留特定字符。普通模板 replace 实际已一致，保留控制场景，没有为其引入无关改动。源码依据为固定 ParseUtil.GetArgumentFromQueryString、StringUtil.MakeValidFileName 与 CardigannIndexer 的 applyFilters。

新增 querystring_filter.go，按原始查询串取值、容错 URI 解码、保留同名首个空值；HTML 属性已在解析层完成 entity 解码，新 HTML 专项确认处理次序。tolower/toupper/validate 共用不变文化映射，保留 dotted capital I/dotless small I；validate 使用明确分隔符和原参数顺序去重。validfilename 依据 Unix/Windows 原生字符语义，空输出返回 `_`；这只是规则字段转换，不替代 Moji 的路径/种子校验。

新增七条正式场景，querystring 含 17 种输入，另有 HTML 属性、模板 replace、validate、Linux 文件名和大小写。隔离后整组实际 110/110（106 正常、4 精确预期拒绝），0 未解释/5 原有解释，报告 `/tmp/moji-indexer-diff-filter-tail-isolated-final.json`。同一参考根并发专项亦通过，18 项比较器、5 项依赖锁检查、完整 Go、cardigann/indexer 完整 race 和相关 vet 通过。两个旧单测对 querystring HTML 解码及 validate 空格的预期经真实上游证据纠正；无新增比较例外、SDK/runner/锁修改。

本轮没有 API/UI/schema/Load 变化，不重复 Web、全库或镜像；Windows 分支没有 Windows .NET 实测，不把 Linux 证据扩大。实际站点、全部非 ASCII 键、其他文化/Unicode 版本和相对日期仍需按 R-004 收尾。

## 相对间隔与固定时钟（2026-11-04）

基线 `0980cca`。读取固定 DateTimeUtil.FromTimeAgo 后识别原 Go 实现将月份/年份按日历加减，与上游 30 天/365 天固定间隔不符。新增明确标记为 relative-date-helper 的独立辅助协议：Go 执行正式 ApplyFilters 的 timeago/reltime，参考程序直接调用未修改的上游 FromTimeAgo(text, relativeFrom)，共用显式子秒/整秒时钟。辅助报告仍逐字比较字符串和输入、失败状态；预期失败在场景中声明，两端不符合声明都会报告错误，未加入浮动时钟容差。

真实红报告 `/tmp/moji-relative-dates-red.json` 复现 2024-03-31 减一个月 Go 得到 3 月 2 日、上游 3 月 1 日，以及 mo 别名被拒绝。第二组整秒红报告 `/tmp/moji-relative-dates-zero-red.json` 进一步复现 float32 月/周/年缩放导致的秒边界差异和 Unicode 空白不被识别。relative_dates.go 已按固定间隔、单精度数值及中间乘法、100 ns ticks、上游单位匹配和无数量时返回原时刻修正；分离整天与不足一天的间隔，避免 Go Duration 对世纪级时间的溢出；保留 DateTime 年份/总量溢出拒绝。fuzzytime 仅其明确 ago 分支共用该间隔，其他日历推断继续独立核对，不将 timeago 的 yesterday 语义混给 fuzzy 日历。

两组 38 个输入（32 合法、6 声明非法）对真实上游函数全部通过，0 未解释/0 例外，报告 `/tmp/moji-relative-dates-final.json`。覆盖闰年、月末、小数日/周/月、浮点取值、混合单位、千位分隔符、Unicode 空白、365 年区间、未知/溢出单位及 now/no-match；不将这些辅助输入计为完整解释器场景。原完整搜索/登录/下载场景仍为 110/110（106 正常、4 精确预期拒绝），0 未解释/5 原有解释，报告 `/tmp/moji-indexer-diff-relative-full-final.json`。

参考新增固定时钟入口，必须重建：最新 `/tmp/moji-indexer-diff-reference-8` 在固定 SDK、原归档及三个依赖锁下 restore/build 成功，31 个 DLL 指纹验证通过；旧 reference-6 被 provenance 拒绝。准备时修正 runner 通过 Value<string> 读取日期 token 会丢子秒的问题，改为保留 DateTime ticks 后在全新目录重建；未改上游解析类。完整 Go、cardigann/indexer/回放 CLI 的 race 与 vet、20 项比较器及 5 项依赖锁检查通过。兼容 CI 追加独立辅助对照步骤及单独报告制品，不增加生产依赖；远端 Actions 尚未执行。

本轮未改 API/UI/schema/Load，不重复 Web、全库静态或镜像；其他 fuzzy 日历、jsonjoinarray 空/非数组及来源/许可等未完成项仍由 R-004 跟踪。没有实际站点、配置、数据库或下载器操作。

### jsonjoinarray 子节点与数值（2026-11-04）

固定 Jackett 实现是 JObject.Parse → SelectToken → 枚举选中 token 的子节点 → JToken.ToString → Join；原实现错误地允许数组根、多 token 展平和标量直接输出，还通过 float64 损失大整数。真实红报告 `/tmp/moji-jsonjoin-probes/extended-red.json` 复现上述差异；后续红报告确认嵌套 JSON 日期及 NEL 转义也不同。现用有序对象、保留数字词法和既有 JSONPath 的 Keyed 接口独立实现：空/null/标量没有子节点，多个匹配拒绝，属性按声明顺序输出，重复属性替换值而不移动位置；嵌套结构使用两空格缩进及 UTC 参考宿主日期表示，保留七位小数和日期种类后缀。没有复制上游代码或新增依赖。

解码限制 64 层/50,000 节点并响应取消；单节点序列化及拼接含分隔符受 MaxTemplateBytes 限制，拼接前计入预算，防止大分隔符导致扩张。正式新增三个合成 HTTP 场景（14 行类型/数值、11 行多选择、3 行嵌套路径），专项和完整实际对照均通过：113/113（109 正常、4 精确预期拒绝），0 未解释/5 原有解释；报告 `/tmp/moji-indexer-diff-jsonjoin-final.json`，参考仍为校验后的 reference-8，无新增比较例外。固定语料只读检查找到两个 jsonjoinarray，均在 amigosshare.yml，用于字符串转义解码及 genres 拼接；未重跑未变化的静态 Load。

本轮完整 Go、cardigann/indexer race 与相关 vet 通过。没有 API/UI/schema 改动，不重复 Web 或镜像；尚未证明全部 JSONPath 方言、非标准 JSON 语法、所有数值/文化/时区组合或真实站点。没有操作实际配置、数据库、站点或下载器。

### fuzzy 日历与日期入口（2026-11-04）

固定规则只读检查发现 21 处 fuzzytime，实际用于 Today/Yesterday、星期及本地化替换后的文本。扩展既有辅助协议，让 Moji 真实过滤器与未修改的 DateTimeUtil.FromUnknown(text, null, relativeFrom) 共用固定 UTC 时钟；按原 SDK/归档/NuGet 锁完整重建 reference-9，0 警告/错误并校验 31 DLL。原 reference-8 不再用于当前 runner。红报告 `/tmp/moji-fuzzy-calendar-red.json` 复现 today/yesterday 错误继承当前时分秒、tomorrow 未识别；绝对日期偏移红报告 `/tmp/moji-fuzzy-absolute-offset-red.json` 复现普通日期被额外换算偏移。

新增 fuzzy_dates.go 按日历语义实现零点、AM/PM、Today/Yesterday/Tomorrow、同周或最近星期、缺年未来日期回退一年及闰日钳制、Unix 秒与范围限制；只有 ago 分支使用间隔解析，裸 1 month 不再被当作合法日历日期。普通绝对日期沿用 DateTimeRoutines 暴露的钟面值，明确 RFC 日期和 day phrase 使用偏移换算；两者不能混为同一解析行为。date 字段与 HTML dateheaders 同样改用该入口，合成搜索回归验证其不会绕过日历语义；不改变业务发行日时区。

新增两组 fuzzy 辅助场景共 44 输入（36 合法/8 声明非法），含 Unicode 空白、时区跨零点、缺年/闰年及最大 Unix 秒；与原两组间隔共 82 输入全部精确通过，报告 `/tmp/moji-fuzzy-helpers-final.json`。独立完整绝对日期搜索/取种场景覆盖 7 行；整组完整解释器 114/114（110 正常、4 精确预期拒绝），0 未解释/5 原有解释，报告 `/tmp/moji-fuzzy-all.json`。没有加入时间容差或新增差异例外；20 比较器/5 锁检查通过。CI 已单独加入辅助组，不将 82 输入冒称完整搜索场景或已运行远端 CI。

最大 Unix 秒的初次完整场景报告 `/tmp/moji-fuzzy-absolute.json` 另确认 BaseIndexer 会把未来发布时间改成 DateTime.Now，而过滤器文字仍保留 9999 年；当前内置来源尚未实现该外层行为，需另行核对修正。该边界保留在精确辅助对照，完整绝对日期场景使用过去时间，不通过放宽比较隐藏问题。FromUnknown 的 ago 分支没有传 relativeFrom，因此辅助组不放入 ago；已有本地 ago 回归不能冒称该分支的固定上游时钟对照。仅时间/库默认日期、其余自由文本和文化/DST 仍不能由当前样本证明。

本轮完整 Go、cardigann/indexer/replay race 与相关 vet 通过。没有 API/UI/schema/Load 变化，不重复 Web/静态全库/镜像，也未操作真实服务或数据。后续先核对未来发布时间，再继续来源/许可与原完整验收范围。

### 未来发布时间的宿主差异（2026-11-04）

后续源码核对纠正“必然属于内置解析缺陷”的判断：固定 Jackett 的 [BaseIndexer.FixResults](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/BaseIndexer.cs) 在非 Debug 模式将未来 PublishDate 改成 DateTime.Now，发生在规则解析之后。固定 Prowlarr 的 [CardigannParser](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannParser.cs)、[IndexerBase.CleanupReleases](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/IndexerBase.cs) 及 [ReleaseInfo](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Parser/Model/ReleaseInfo.cs) 这条解析/清理路径保留解析值，没有对应改写。

已按项目默认行为复核要求询问用户：沿用 Jackett 钳制为检索时间，还是沿用 Prowlarr 保留原解析时间。尚未收到选择，未更改该默认行为、放宽时间比较或删除原失败报告。此项单独等待选择，其他验收继续。

### 来源与镜像许可文本（2026-11-04）

重新读取固定 Indexers 归档，未找到独立 LICENSE/COPYING/NOTICE；当天官方 [GitHub 仓库元数据](https://github.com/Prowlarr/Indexers) 的 license 为 null，固定 README 说明自动同步 Jackett 定义。实际核对固定 Jackett GPL v2 和 Prowlarr GPL v3 根文本；这不自动建立每份 Indexers 文件的独立授权，也不选择 Moji 的发行许可证。内嵌 schema 与归档逐字节 SHA-256 相同：`376976d75daf036ead90dc55bf542ce602da8d24deb31cc10200b6bf181e3e8e`。完整事实/来源更新在 [NOTICE](../../internal/cardigann/NOTICE.md)；应用发行许可、schema 再分发授权及完整依赖许可审查仍未完成，没有对其作法律结论。

发现并修正具体打包遗漏：编码表的 .NET MIT 全文虽在源码中保留，原运行镜像没有复制。Docker 现包含 `/usr/share/licenses/moji/cardigann/NOTICE.md` 与 `DOTNET-LICENSE.txt`，smoke 对两文件逐字节 SHA-256 核对；未改变许可正文或新增应用许可证。独立候选 amd64 镜像 `sha256:f350d784f1ba460a10238b713e405880c5a9bdca0df32a0571666c22a8dd4543` 构建及完整断网 smoke exit 0，版本为 `dev-53210917d6c8-dirty`，源码为 5321091 加本批改动，不冒称已提交冻结镜像。日志 `/tmp/moji-indexer-notice-build.log`、`/tmp/moji-indexer-notice-smoke.log`；shell 语法、12 版本/12 发布脚本测试、文档和 diff 检查通过。

该 smoke 同时覆盖非 root、媒体工具链、无 Jackett/无规则库启动、Home 本地读取、配置保存、优雅退出和容器替换后的三库保留，但不覆盖已安装规则、真实站点或 ARM64。没有运行生产实例、推送或部署；最终冻结提交后的多架构镜像仍须另验。

### 旧版升级、请求版本与离线回退（2026-11-04）

从基线 502cfa9 的 git archive 独立编译旧应用、旧 moji-storage 及只写合成任务的旧 TaskStore fixture；应用版本 `upgrade-old-502cfa9`。新程序基于 ed28f06 加本批修复，标为 `upgrade-new-ed28f06-working`，不冒称已提交冻结版本。所有容器断网、只读根，只挂专用 `/tmp/moji-indexer-upgrade-proof` 与测试二进制目录；旧程序启动要求的 Dashboard 登录由同容器 loopback 假服务应答，没有真实 Jackett、qB 或 Stash。初次合成配置使用了不支持的 en-US，修正为 en 后重跑；旧程序有密码时强制登录的启动失败也保留日志，未据此修改生产旧代码。

实际旧版创建 core v3/runtime v1/cache v1 后，旧 TaskStore 保存包含普通及 UUID 形状 Jackett ID 的原始 DownloadRequest，并由旧 storage 工具保存带配置的受保护备份。新 Manager 逐序升级为 core v6/runtime v6/cache v1；旧 ledger 前缀（名称和 checksum）完全一致，任务主字段及请求 JSON 字节完全一致，未重建或改写历史迁移。原 v2 配置及无关 custom 值、所有模拟凭据保留；离线工具验证 v2→v3 与再次 v3 保存有效值一致，原 v2 文件逐字节不变。

升级复现两项实际缺陷：配置工具首轮因 UUID 裸 ID 被当成内置实例而拒绝保存；任务原函数探针将同一旧 ID 发送为 builtin UUID，且 version 99 也会发起搜索。红报告 `/tmp/moji-sourcing-version-red.log`、`/tmp/moji-indexer-upgrade-proof/config-migration.log`。修复按原配置版本迁移 Jackett-only 身份，普通设置保存保留已解析的来源资格，旧设置命令仅在显式改变顺序时按已配置内置身份解析；重复保存、局部 patch 和既有排序保存回归通过。

新增 sourcing_request codec：新写入 version 1，持久保存明确来源 ID、保留 false/true/缺省下载选项，64 KiB 预算；旧无版本请求一律解释为 Jackett-only，UUID 不例外，未知版本与非法新格式在搜索前阻塞并保留原请求。队列、资源跟踪、手动候选接收/替换均统一；运行时不再凭字符串猜已保存来源，遗留 tracker 接口单独去除已明确的 Jackett 资格。数据库 checkpoint payload_version 与此 JSON 的 version 是不同层级，本批不修改已应用迁移或公共 API。

新应用在迁移后的库及 v3 配置下实际启动，GraphQL 确认 Jackett 仍启用、没有内置实例或活动规则库，优雅退出 exit 0。旧应用拒绝 v3 配置且没有打开数据目录；旧 storage 拒绝 v6，三个数据库摘要不变。另在独立副本添加连续的未来 core v7，当前 storage 也拒绝且核心库摘要不变。用旧 storage 工具离线恢复旧受保护备份，配置字节相同、请求 JSON 字节相同，回到 core v3/runtime v1/cache v1；旧应用重新启动、GraphQL 版本及退出 exit 0 均通过。新版本 restore 会按其迁移表升级，不能用它冒称生成了旧版可读数据库。

完整证据、二进制 SHA-256 和报告保留在 `/tmp/moji-indexer-upgrade-proof/report.json`、`old-state.json`、`upgraded-state.json`、`app-replay.log` 及对应拒绝/恢复记录。源码/编译和仅测试 fixture 都在临时目录，未把旧配置、数据库、日志或实际站点规则提交到仓库。本轮完整 Go、config/taskruntime/taskflow/config CLI race 与相关 vet 通过；没有引擎/schema/API/UI 改动，不重复此前无变化的差分/Web/镜像。已安装规则的回滚和资产恢复仍由原独立测试证明，不能把本次旧版无规则库演练扩大成全部场景；多架构冻结镜像、真实站点/代理/solver、来源/许可仍按 R-004 继续。

## 已授权八站真实搜索与资源获取（2026-11-04）

新增显式 opt-in 的 [验收入口](../../cmd/moji/indexers_live_acceptance_test.go)，正常离线 suite 跳过；参数与边界见[开发流程](../workflows/development.md#索引器真实验收显式-opt-in)。按用户授权只读根 config.yaml、复制到私有临时配置/DB，调用正式 indexer runtime、indexercheck 与两类 adapter，固定 5f5db4e 的 579 定义，关键词 SSIS；补充 u3c3 的 SONE 查询。没有组装或调用 qB/Stash，所有完成报告确认 tasks=0、原配置字节不变。报告为新建 0600 文件，不允许覆盖已有文件；已用私有配置副本证明防覆盖失败发生在联网前且文件摘要不变。

本地 Jackett 镜像 `sha256:697911d750b823e9a7eaac0511dba84460e2c174a72923eca6da1a3dd5360362`，Linuxserver v0.24.2830-ls3；FlareSolverr 镜像 `sha256:680575c113318c2baa22cd507d3fb678a45799ac39af3c54b1f2ca9e83006418`。这是现场版本，不是合成对照的固定 Jackett SDK 版本。原配置没有 Moji solver/代理网络配置；先测默认直连，再在临时副本为挑战站点启用已有 localhost:8191 solver。没有开启/修改真实 Jackett 站点配置或重启服务。

| 配置站点 | 内置实际结果 | 外部 Jackett 实际结果及边界 |
| --- | --- | --- |
| freejavtorrent | 20 条，获取及 ValidateArtifact 通过 | 20 条，获取验证通过 |
| sukebeinyaasi | 20 条，获取验证通过 | 20 条，获取验证通过 |
| 0magnet | 20 条，获取验证通过 | 首轮获取 30 秒请求超时；完整预算复测后 20 条及获取通过，属于观察到的波动 |
| tokyotosho | 默认挑战页；临时启用 solver 后 20 条及获取通过 | 搜索约 30 秒请求超时；正式总预算复测仍同样超时 |
| onejav | 默认挑战页；solver 后搜索 20 条，获取阶段仍遇挑战，按既定策略不委托/重放取种链路 | 搜索 20 条；首轮 45 秒总预算、复测正式总预算下获取均超时，后者明确为单请求约 30 秒 |
| u3c3 | SSIS、SONE 都为空，没有取种样本 | 两关键词也均为空；不能据此证明真实下载或站点所有查询可用 |
| 52bt | 固定规则默认地址返回 HTTP 410，尚未进入结果解析 | 搜索失败；其配置 origin 与固定默认地址不同，不能算相同页面的解析差分 |
| torrentkitty | 守卫传输失败、无 HTTP 响应；独立直连 curl 也在 TLS 握手前失败 | 不在当前 Jackett 配置目录中，配置阶段 SCOPE_INVALID，没有执行搜索 |

首轮 16 检查使用了额外 45 秒测试总预算；已纠正最终入口沿用正式三分钟预算，对三个超时 Jackett 路径独立复测，没有把人为预算当成产品故障。其余补充为两站 solver 与 u3c3 双路径，共 23 次阶段检查；这是观测数量，不是通过数。三个默认内置站点完成搜索/获取，加入 solver 后新增 tokyotosho；onejav 的取种挑战仍是明确限制。solver 时间窗私有日志含两个目标为 SSIS、host 匹配 onejav/tokyotosho 的 request.get，记录两次检测和求解成功；随后原 HTTP 搜索验证通过，未发生取种自动求解。

私有证据 `/tmp/moji-live-indexers-20261104.json`、`/tmp/moji-live-jackett-full-budget-20261104.json`、`/tmp/moji-live-solver-20261104.json`、`/tmp/moji-live-u3c3-20261104.json`；仅临时原始响应诊断保留在 `/tmp/moji-live-response-diagnosis-20261104.json` 及 0600 的 body 文件，不纳入仓库。目录 origin 核对确认除 52bt 不同、torrentkitty 未配置外，其余六站与规则默认 origin 相同；没有冻结 Jackett Cookie/规则/代理或实时内容，不能将该表冒称所有配置完全相同的可重复差分。没有真实账号登录或显式转发代理验收，公开/JSON/中间页/CF 会话能力仍分别结合固定 fixture 与当前实测判断。

官方当前 v11 的 52bt 已更换镜像地址，提示本次固定语料并不代表社区最新修订。下一步验证真实官方规则更新及独立回滚，不为 52bt 写专用解析器，也不把固定地址 HTTP 410 归因为 Moji 正则或 JSON 解析缺陷。本批只增加验收入口/文档，完整 Go/vet 默认离线执行通过；没有修改生产 API/UI、规则或解析器，未推送或部署。

## 真实官方规则更新与独立回滚（2026-11-04）

新增默认跳过的 [官方更新入口](../../cmd/moji/indexers_live_rules_test.go)。原配置只读、状态和设置写入临时副本，无下载器装配。以固定 `5f5db4e58d06be18619c3a858b9a91ade6f341ca` 归档引导后，关闭并重建索引 runtime，使用实际默认 GitHub API/codeload 客户端。实际两次均发现并激活 `9ecfe4f9833d3394b27fe911a9a6242b3a9da90d`：仅检查保持旧活动版本；下载 745017 字节，580 个定义全库运行时校验通过；回滚旧修订完成，下载字节为 0，新修订保留为 previous。没有替换程序，固定回归语料仍为 579 定义的旧修订。

报告 `/tmp/moji-live-official-rules-20261104.json` 与 `/tmp/moji-live-official-rules-solver-20261104.json`，同名日志均 exit 0。52bt 链接确实随社区规则更新，但搜索仍未通过：直接路径为 `INDEXER_CHALLENGE_REQUIRED`；显式配置已有 solver 的复测为 `INDEXER_OPERATION_FAILED`，不能据此断言这次 solver 调用成功。两次均无候选、未获取资源，不能把生命周期成功计为 52bt 可用。原配置字节不变，临时核心 tasks=0。两份既有报告先于 SolverConfigured 字段补充，该次 solver 配置由实际命令记录证明。

完整 Go 和 cmd/moji vet 通过，默认离线入口跳过。最终测试报告将未查询任务数初始化为 -1，避免早期失败误报零任务；报告必须是新的私有文件。显式运行成功仅表示完成安全观测，各站失败仍保留。来源检查、更新及回滚均为测试临时状态，不涉及生产规则或服务变更。

## 五种声明式登录方法对照（2026-11-04）

初审表中缺少实际差分的 get/oneurl，连同 cookie 补为三条[正式合成场景](../../scripts/indexer-diff/cases/login-get/definition.yml)，各自完成认证验证→搜索→获取 torrent。GET 含空格、加号与 ampersand 编码，oneurl 验证路径直接拼接，cookie 不发登录提交；请求、Cookie、结果与资源全部精确比较。固定 reference-10 专项 3/3 无例外通过；没有修改生产实现、参考程序、比较条件或旧场景。

完整实际解释器对照 117/117（113 正常、4 精确预期拒绝），0 未解释/5 原有解释差异，报告 `/tmp/moji-login-all-20261104.json`、专项 `/tmp/moji-login-methods-20261104.json`；20 项比较器测试与 replay 包测试通过。加上原有 form/post，五种 method 均有实际执行样本，不等于每种站点/重定向/错误组合全部认证，亦不替代真实账号登录验收。没有真实凭据、网络或生产下载器操作；默认离线 replay 测试与 CI 自动收集这些 fixture。

## 声明式登录默认值（2026-11-04）

审查固定 Jackett 构造器与 Prowlarr 的 Cardigann 设置投影，发现合法省略 settings 的定义未获得默认 username/password，Moji 会拒绝配置并无法登录。独立实际差分 `/tmp/moji-default-settings-before-20261104.json` 先复现 `unknown rule setting username`，参考程序已完成取种。另一个独立差分 `/tmp/moji-default-method-before-20261104.json` 复现省略 login.method 时 Moji 错发 GET，而固定 Jackett 默认 form。

在生产 Load 统一补齐默认字段及 form 方法，目录/配置校验/执行读取同一个有效定义，原始字节 digest 不变；显式 settings: [] 保持空并拒绝隐式凭据。密码字段保留 password 类型。两个专项合成场景均认证验证→搜索→获取通过，method 场景包含原文字挑战，报告 `/tmp/moji-default-login-after-20261104.json`。没有修改原 schema、参考程序或比较例外；默认值范围与真实站点可达性分别解释。

完整 Go、cardigann/admin/indexer/replay race 和相关 vet 通过；固定库 579/579 静态编译、1 个既有警告、0 失败，报告 `/tmp/moji-default-login-v11-20261104.json`。完整实际差分 119/119（115 正常、4 精确预期拒绝），0 未解释/5 原有解释差异，报告 `/tmp/moji-default-login-full-20261104.json`。本次无需 schema/API/UI 生成变更，无生产配置/实例/下载器操作。正在构建的 afabfb4 ARM64 镜像读取的是修复前源码，不包含本修正，不能作为本修正的镜像证据。

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

### fea8176 干净源码镜像（2026-11-04）

干净已提交 `fea81765e89f262dd694b97c4647920bb67a8ac7` 实际构建 amd64 镜像 `sha256:aa03d44596cb68a9ba2acbe0e2da116ed4a9dfb9c28979240c8649a7cbd94946`，版本 `dev-fea81765e89f`、OCI revision 与源码一致。构建及完整断网 smoke exit 0，日志 `/tmp/moji-indexer-fea8176-amd64-build.log`、`/tmp/moji-indexer-fea8176-amd64-smoke.log`。包含截至该提交的引擎、许可文本打包与来源迁移修复，覆盖非 root/只读根、媒体工具链、版本一致、NOTICE/MIT 内容摘要、无 Jackett/规则库启动、Home 本地状态、保存/优雅关闭及替换容器后三库保留；只使用临时容器和卷，没有推送或部署。

同一提交再次运行固定归档的生产 Load/static compile，579/579、1 个已知可选选择器警告、0 失败；报告 `/tmp/moji-v11-fea8176-report.json`。当前未发现专用 live/acceptance 环境参数或配置，已请求用户提供允许的站点/搜索与取种范围及本地测试路径；没有将现有生产凭据当作测试授权。QEMU 用户态二进制仍不可用，未把 amd64 或交叉编译当作 ARM64 运行验收。此镜像不包含之后的代码改动；静态规则兼容、镜像 smoke、真实站点与 ARM64 继续分别报告。

后续用户已明确授权参考根目录 config.yaml、按其中站点执行真实搜索并获取种子，不向生产下载器提交任务。离线配置工具只读验证 exit 0；保护日志 `/tmp/moji-live-authorized-config-validation.log` 不纳入提交。真实请求尚未开始，不把配置有效当作站点可用证据。

### b91ed24 干净源码镜像（2026-11-04）

已提交 `b91ed24419c96b14f702ec103877e0b3bfd65f48` 的独立 amd64 镜像构建成功，镜像 `sha256:08640ea862bd51c6ac1f2a2f4cd712491aa7dec9f0e754adb01a421477806b6a`，版本 `dev-b91ed24419c9`、OCI revision 一致，包含仅时间日期修复。构建日志 `/tmp/moji-indexer-b91ed24-amd64-build.log`。首次 smoke 启动时主工作区已新增未提交 fixture，版本脚本因此选择 dirty 标签而未找到对应镜像，未执行验收；随后在 `/tmp/moji-indexer-b91ed24-frozen` 干净隔离检出运行原 smoke，完整通过，日志 `/tmp/moji-indexer-b91ed24-amd64-smoke-final.log`。

该次覆盖镜像内后端/Web/OCI 版本、非 root/只读根、媒体工具、NOTICE/MIT 摘要、断网无 Jackett/无规则库启动、本地概况/无隐式检查、配置保存、优雅关闭和替换容器后三库及设置保留。仅操作独立临时容器与卷；没有推送、部署或修改生产实例。ARM64 仍未取得可执行环境，不能由此推断通过。

### 仅时间日期（2026-11-04）

固定库 1337x.yml 的 date_today 确有仅时间用法（12:25am），随后通过 .Result.date_today 写入正式 date，失败会回退 now。已用本地回归及实际 reference-10 复现拒绝。辅助模式给未修改的 DateTimeRoutines 设置其公开 DefaultDate，以固定无日期输入的默认日；不改完整解释器模式的宿主时钟。全新固定 SDK/源码/NuGet 锁构建 reference-10 成功，0 警告/错误、31 DLL 校验；旧 reference-9 对当前 runner 不再有效。

当前代码已独立补仅时间钟面日期、AM/PM、偏移保持与整数秒；19 输入（14 合法/5 声明非法）与上游实际对照通过，报告 `/tmp/moji-time-only-upstream-fixed.json`。已执行的 cardigann/replay 包测试通过，新增字段→正式发布时间的本地回归也通过，日志 `/tmp/moji-time-only-local-final.log`。`make generate-moji-graphql` 实际运行成功且未产生生成文件差异，日志 `/tmp/moji-indexer-final-codegen.log`。

初次完整验证曾因自动审批审查用量限额未执行；在提示的恢复时间后通过正常审批继续，没有绕过。最终完整 Go、cardigann/indexer/replay race 与相关 vet 通过；完整解释器仍为 114/114（0 未解释/5 原有解释），五组共 101 输入（82 合法/19 声明非法）精确通过，20 比较器/5 锁检查通过。报告 `/tmp/moji-time-only-full-final.json`、`/tmp/moji-time-only-helpers-final.json`。CI 加入第五辅助组；辅助输入不计为完整搜索场景，未放宽时间比较。仅时间结果字段的本地搜索回归确认不会静默回退为 now；没有 API/UI/schema 变化，不重复未受影响的 Web 或全库静态检查。fea8176 镜像不包含该修复，后续需重新冻结源码镜像；其他自由文本/DST 未由本轮证明。

### select 空值边界修正

继续验收时，固定实际 Jackett 的两条红报告均证实：Moji 在没有默认值或用户明确配置空字符串时，错误地选择排序后的第一个选项并加入请求。删除引擎与管理投影的这一回退；保留非空未知选项拒绝。编辑器显示“未选择”，不把浏览器原生第一个选项误表示为当前值；清除已配置 select 后显示规则声明默认值，提交仍仅发送 clear。

新增两条正式合成场景，四种 select 默认/覆盖均无例外通过。完整差分现为 87/87（83 正常、4 精确预期拒绝），0 未解释/5 原有已解释差异，报告 `/tmp/moji-indexer-diff-select-final.json`；未改参考 runner 或比较容差。完整 Go、相关 race/vet、Web 编辑器 7 项、i18n 10 项及生产构建通过；固定 Playwright 容器 indexers.spec.ts 36 项无更新/无重试通过，桌面/手机空值截图人工核对、无横向溢出或浏览器/GraphQL 错误。此修正发生在上述 `0a34d43` 镜像之后，该镜像不包含本修正；没有操作实际配置或站点。
