# Cardigann v11 请求语义复核（实施中）

范围：搜索参数继承、空值、GET/POST 去重、`$raw`、查询与表单字符编码。不是完整兼容或实时站点验收。

## 固定证据

- 规则语料：Prowlarr/Indexers `5f5db4e58d06be18619c3a858b9a91ade6f341ca`，归档 SHA-256 `b266646ab2662c260ac6b0fad5f99794b200784f3ac865b521820e8e6814a472`。
- [Prowlarr 请求生成器](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannRequestGenerator.cs)：全局 inputs 后接路径 inputs；搜索 raw 的两阶段编码；仅 GET 去重。该快照包含较新引擎，不能据此把 v12 特性加入 v11。
- [Jackett Cardigann 实现](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/Definitions/CardigannIndexer.cs)：交叉核对上述请求行为。
- [Prowlarr 查询序列化](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Parser/StringUtil.cs)与[编码扩展](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Common/Extensions/StringExtensions.cs)：值先按规则编码取字节，再进行表单编码。
- [.NET WebUtility](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Private.CoreLib/src/System/Net/WebUtility.cs)：安全字符为字母、数字和 `-_.!*()`，空格编码为 `+`，`~` 需要编码。
- [.NET 编码别名表](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Text.Encoding.CodePages/src/Data/CodePageNameMappings.csv)：TIS-620、ISO-8859-11 对应代码页 874；不采用 HTML 编码别名替代查询编码。

## 已修正的行为与回归见证

- 全局与路径参数使用有序的参数组，重复名称都发送；不再用 map 覆盖。继承关闭仅执行路径组。GET/POST 均有回归样本。
- `allowEmptyInputs: false` 排除仅含空白的普通值；允许空值时仍保留。
- 相同 GET 路径只请求一次；明确声明的相同 POST 路径逐个执行。没有增加自动重放。
- 搜索 `$raw` 先以 UTF-8 WebUtility 规则编码模板替换值，再按 `&` 与首个 `=` 拆分，最后按声明编码序列化值。保留重复参数、空值和值中的等号，忽略空键。通用请求不把 `$raw` 当成搜索专用语法。
- 普通搜索查询与 POST 表单、urlencode/urldecode 使用声明的编码；增加 Windows-1251、ISO-8859-1、泰文 874、表单保留字符和语料全部编码标签样本。ISO-8859-1 的控制字节不被 HTML 别名映射为 Windows-1252。
- 通用 `download.before` 保留空字符串和仅含空白的参数；搜索的 allowEmptyInputs 行为保持独立。GET/POST 的准备请求→最终种子流程均先复现空参数丢失，再修正；断言最终字节和两次请求，无额外取种。
- 样本见 `internal/cardigann/search_test.go`、`query_encoding_test.go`；CLI 特征报告测试见 `cmd/moji-indexer-check/request_features_test.go`。

## 旧编码回退与独立运行时对照

v11 语料的九个旧编码标签对应八个代码页。`sbcs_encoding.gen.go` 现在包含官方 .NET 9 round-trip/best-fit 小表，补齐严格 Go 编码器缺失的近似与替代行为；查询/表单及 URL 过滤器使用同一 codec。约 18 KiB 打包映射，没有给 Moji 增加 .NET 运行时依赖，也没有内嵌社区站点规则。其他 IANA 编码仍走 x/text，不以这些样本认证它们。

来源锁、SHA-256 和离线生成工具见 `scripts/cardigann-encodings/`。数据取自[官方 codepages.nlp](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Text.Encoding.CodePages/src/Data/codepages.nlp)与[Latin-1 best-fit 表](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Private.CoreLib/src/System/Text/EncoderLatin1BestFitFallback.Data.cs)，MIT 原文保存于 `internal/cardigann/DOTNET-LICENSE.txt`。代理对 .NET 内部回退行为的实现依据包括[SBCS 编码器](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Text.Encoding.CodePages/src/System/Text/SBCSCodePageEncoding.cs)和[回退器](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Text.Encoding.CodePages/src/System/Text/EncoderBestFitFallback.cs)。

使用固定官方 SDK 9.0.100 镜像 `sha256:6125486580cdf529a2a337d51c5557af08eaa04ef08b04f5b1d5a94f6bec7f1a`，确认实际运行时 9.0.0；容器无网络、根文件系统只读，仅挂载临时项目。生成八页全部 BMP UTF-16 输入编码、全部 256 字节解码的 SHA-256，以及代表性样本。默认 Go 回归独立核对这些摘要；接入前八页全部报 unsupported rune，接入后全部匹配。包括 ē→e、全角字母数字→ASCII、不可表示字符→?、有效代理对→??，并验证 UTF-8 分块和输出缓冲不足时不会丢字节。

`check_reference.py` 已从仓库重新执行，八个参考结果匹配；离线生成 --check、全量 Go、cardigann/indexer/defs/check/cmd 完整 race/vet 均通过。固定语料再次 579/579 静态编译、1 个警告、0 失败。这是编码算法的真实 .NET 可执行对照，仍不是 Jackett 完整索引流程差分。

## 可复查的语料统计

`check_corpus.py --archive /tmp/moji-indexers-v11.tar.gz --report /tmp/moji-v11-request-audit.json`：579/579 静态编译，1 个定义警告，0 失败。增强报告包含 `requestFeatures`、`requestDefinitions`、`encodings`、`queryModes`、`queryVariables`。

- `$raw`：416 条搜索路径，涉及 389 个定义。
- POST：12 条搜索路径，涉及 12 个定义。
- 非 UTF-8：42 个定义，10 种编码标签总计覆盖全部语料。
- 这份语料未发现输入继承关闭或跨组同名键；这些行为由契约回归覆盖。统计与静态编译不能证明路径实际成功。

## 响应解码与安装期编码校验

先以完整搜索路径复现：ISO-8859-1 字节 0x80 被 HTML 别名解读成欧元符号；Windows-1251 XML 的 Тест 被再次转码为 РўРµСЃС‚；未知编码未在加载期拒绝。

共享 `declaredEncoding` 现在供查询、URL 过滤器和响应解码使用。响应使用声明的 codec，UTF-8 也经过显式解码，以保留 BOM 并替代非法字节；仍在解码后执行 16 MiB 字节预算，验证了输入未超限但解码展开超限的拒绝。搜索、登录与取种页面原本均调用同一 responseText，没有新增第二套解码路径。

XML 接收到的是已解码 UTF-8 文本，CharsetReader 不再按原始编码声明重新转码。依据是 Prowlarr [HttpResponse.Content](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Common/Http/HttpResponse.cs) 优先 Request.Encoding.GetString，以及上游[解析器](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannParser.cs)以字符串调用 AngleSharp XmlParser。

安装期未知编码返回 `/encoding` 与 YAML 行列诊断。除加载与 staging 拒绝样本外，完整规则更新作业已验证返回 DEFINITION_BUNDLE_INCOMPATIBLE、不发布坏候选、保持旧版本可用。全部使用临时存储与假归档。

`response_encoding_test.go`、definitions bundle/activation 回归通过；全量 Go、cardigann/indexer/defs/check/cmd 完整 race/vet 通过，固定语料再次 579/579 编译、1 个定义警告、0 失败。报告 `/tmp/moji-v11-response-audit.json`。没有把源码依据或本地流程样本冒充真实 Jackett/站点验收。

## 类别输入、路径选择与结果映射

`cardigann.Query.Categories` 改为原始数字 Torznab ID，`.Query.Categories` 保留它们；`.Categories` 才是当前路径的站点类别。标准类别表从 indexer 提取到 `internal/indexercategory`，由适配器与引擎共用，避免两份映射或不必要的数字→名称→模板转换。公开 Moji 查询仍使用原有数字 ID，配置和持久格式不变。

先复现再修正：父类别展开后重复站点 ID 没有去重；取反路径符合条件时错误保留非排除类别；结果 ID 大小写差异导致映射丢失；重复 categorydesc|append 拼接描述而非合并类别；noappend 未映射到新类别时清空已有类别。

依据是 [Prowlarr 类别能力映射](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/IndexerCapabilitiesCategories.cs)的父子类别展开、Distinct 与大小写无关匹配，[请求生成器](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannRequestGenerator.cs)和[Jackett](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/Definitions/CardigannIndexer.cs)的取反/交集行为，以及 [Prowlarr 结果解析](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannParser.cs)仅在新映射非空时替换、否则做集合合并。

空查询且无默认映射时执行未限制的类别路径，沿用现代 Prowlarr。这里与 Jackett 在 mappedCategories 为空时仍检查正向路径的实现不同，已用 unrestricted 样本固定选择；不宣称两者逐字节完全相同。大小写目前使用 Go Unicode folding，没有认证 .NET 所有非 ASCII 文化比较。类别列表的源顺序已在后续阶段修正，见下节。

语料含 56 条带类别约束路径，未发现取反路径或直接引用 `.Query.Categories`；其契约由合成规则见证。字段扫描仅发现 hdonly-api.yml 的一次 categorydesc|append：它是第一次解析 categorydesc，不能据此声称该站点发生重复拼接；重复追加问题由独立合法 fixture 复现。

`category_search_test.go` 覆盖原始 ID、父/子类别、去重、正向/取反、默认与未限制搜索；`category_result_test.go` 覆盖大小写、追加、无效/有效替换。BuiltinAdapter 实际组装链路同时验证父/子原始 ID、站点 ID、会话重建、过期重认证及单次取种。最终全量 Go、cardigann/indexer/defs/check/cmd 完整 race、相关 vet、579/579 语料编译通过（1 个已有定义警告、0 失败）；报告 `/tmp/moji-v11-category-audit.json`。旧 36349 句柄恢复时已不存在，未据此假定成功；以最后 67553 全部命令 exit 0 为准。

## YAML 源顺序与登录表单

先用字节级样本复现：search inputs 按字母排序，把 `$raw` 移至其他参数之前；类别 9、2 被改为 2、9；登录 POST 用 Go 默认 UTF-8 编码及不同的表单安全字符；表单重复 name 保留多值，字段与选择器顺序被排序。

`ScalarFields` 现在以有序标量条目保存 request/search/login inputs 和 caps.categories；登录 selectorinputs/getselectorinputs 复用有序 Fields。规则是只读计划；运行时覆盖值使用私有 formFields 字典，以位置索引维护插入顺序和 O(1) 更新，覆盖保留首次位置，重复 DOM name 使用最后值。搜索参数组仍按全局→局部拼接，`$raw` 在声明位置展开；匹配类别去重时保留首次顺序，默认类别顺序也保留。

Prowlarr [请求生成器](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannRequestGenerator.cs)按字典枚举顺序构建参数，表单用 pairs[name]=value 覆盖；[HttpRequestBuilder](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Common/Http/HttpRequestBuilder.cs)按 FormData 顺序、声明 Encoding 序列化值。登录 POST 与验证码续提交现在使用同一声明编码和 WebUtility 表单安全字符。GET selector inputs 按上游默认 UTF-8 序列化，保留已有查询的字节/顺序并覆盖同名键；Prowlarr 拼接二次问号的行为未采用，保留 Moji 原有的有效查询合并语义。

`request_order_test.go` 覆盖 GET/POST、继承重复键、raw 展开及类别/默认类别顺序；`download_test.go` 验证前置 GET/POST 原始字节；`login_order_test.go` 覆盖 GET/POST 旧编码、安全字符、DOM/声明/选择器顺序、重复覆盖、已有查询与验证码单次续提交。全量 Go、cardigann/indexer/defs/login/check/cmd 完整 race 与相关 vet 通过；固定库 579/579 静态编译、1 个已有定义警告、0 失败，报告 `/tmp/moji-v11-order-audit.json`。不改变用户配置或原始规则摘要，无 .NET 运行时依赖，没有真实站点实测或提交部署。

## 真实 Jackett 差分工具（首批执行覆盖）

新增 `scripts/indexer-diff/` 与 `cmd/moji-indexer-replay`。参考程序直接引用固定 Jackett 源码的完整 CardigannIndexer 类，不重写其解释算法；参考入口依据[上游 HTML 测试](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Test/Common/Indexers/CardigannIndexerHtmlTests.cs)。HTTP 客户端保留上游准备逻辑，以 Run 替换为合成响应分派，并调用其 protected 表单编码器获取 POST 字节。Moji 仅使用假 RoundTripper，没有 DNS/socket 回退。

`upstream.lock.json` 固定 Jackett `cfdc645711c487d05cb55932e394b2370154d81e`、官方归档 SHA-256 `461dbcb4ea06b8e69d22164948be0886acb9a2f925f6c74c65c89bf217664540` 和 SDK 镜像摘要。仓库构建脚本已在全新临时目录成功执行：源码不改动，31 个输出 DLL 与 runner 源码摘要进入 provenance；后续比较验证摘要。恢复公共 NuGet 包的构建阶段联网，实际执行禁网、根文件系统只读，仅挂载临时源码/合成场景。NuGet 传递解析尚未独立锁定，以已执行 DLL 摘要留证，不能声称全部依赖可无差异重建。

首批 public-html、public-json、post-login 均以同一 YAML/响应执行搜索→取种；POST 场景包含配置、登录、Cookie 验证、搜索与下载。最终报告 `/tmp/moji-indexer-diff-final.json`：3 PASS、0 未解释差异，结果和种子 SHA-256 一致。请求原始 URL/body/headers 均保留；只规范头名称与等价 UTC 小数格式，精确默认 Accept/Accept-Language 单列解释。自定义头、URL/正文差异不忽略；有意差异要求精确路径/双边值/原因，过期预期失败，执行错误不能作为成功。比较器 9 项测试通过。

真实差分发现直接取种未提供详情页或规则 Referer 时缺少默认来源。已先复现再补为取种目标，现有详情页来源和明确规则头保持；离线重放与规则显式 Referer 回归通过。这是首个真实引擎对照发现并修正的流程差异。

主 CI 默认跑全部 Go 与比较器测试；可重用 indexer-compatibility workflow 构建参考、执行比较、下载/编译固定语料并上传报告，publish 依赖它及 Web/Go/Docker。YAML 已本地解析并验证调用与发布依赖；尚未提交/推送或运行远端 Actions。最终完整 Go、cardigann/indexer/defs/replay/cmd 完整 race/vet、两个旧 Python 脚本各 12 项、固定 579/579 编译通过（1 个已有定义警告、0 失败）。没有改变真实配置、数据库、站点、任务或下载器。

边界：HTTP adapter 层对照不认证实际 TLS/代理/浏览器；Jackett 仍用实际时钟，首批避开相对日期，只固定 UTC。中间下载页、人工挑战、其他模式、错误/取消与全部语义仍需扩展。不能以 3 个合成用例宣称全库实际站点验收完成。

## 中间下载页与前置请求差分

新增 download-page、download-before、tested-download：同一 JSON 搜索结果指向详情地址；两边分别执行 HTML 中间页或 POST before，再解析最终链接并取种。先保留原始失败报告 `/tmp/moji-indexer-diff-downloads-red.json`：结果与种子均成功，差异分别是最终 Referer，以及 testlinktorrent 的请求次数。

这轮没有通过修改生产逻辑来凑齐请求：已有 Moji 选择器取种保留原详情页 Referer，Jackett 改为最终文件地址；Moji 复用已验证的种子，Jackett 先验证后再下载一次。三个场景各记录精确的双边值与理由；tested-download 对完整请求序列留证，额外 URL、正文、请求或头变化都会使预期失效。最终结果、种子摘要不属于该例外，仍严格比较。

最终 `/tmp/moji-indexer-diff-downloads-final.json`：6/6 PASS，0 未解释差异，3 项已解释差异；六个场景的种子 SHA-256 均为 `05453e95d2e4b252277e10495ec01480f4a0c584476acaa335b8d57a83805216`。控制台也显示已解释数量，不把它们写成完全无差异。比较器增加“接受请求差异仍不能掩盖 artifact 变化”回归，并固定六个必需场景，10 项 Python 测试通过。文档与 diff 检查通过；本轮仅改合成语料和比较器，未重复不相关 Web/Go 全量验证。没有真实站点、配置、数据库、任务、提交或部署操作。

人工挑战、其他模式、错误/取消和相对日期仍未成为可执行差分覆盖；本节不把中间页扩展当成整个阶段 E 完成。

## 结构化查询与 TV 派生变量

新增电影（Year/IMDB）、音乐（Artist/Album）、图书（Author/Title）与 TV 季/集/日期场景。先由真实 Jackett 复现五类 TV 请求差异：当前引擎仅透传 Season/Ep，没有生成 EpisodeSearchString，Keywords 缺少 S01E02、S03 或每日日期；零季被错误保留为 season=0。

`query_episode.go` 现区分原始 `.Query.Ep` 与派生 `.Query.Episode`，规范化季字符串并加入最终 Keywords；整季、每日日期、零季以及数值/格式边界有离线完整请求回归。依据为 [Jackett TorznabQuery.GetEpisodeSearchString](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Models/TorznabQuery.cs)、[ParseUtil](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Utils/ParseUtil.cs)及 [Prowlarr TvSearchCriteria](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/IndexerSearch/Definitions/TvSearchCriteria.cs)。未改变配置、公开 API 或任务持久格式。

扩展实际用例还纠正了两次初版数值推断：01A 经 CoerceInt 成 1；单一小数点会先转为分组分隔符，所以 1.5 成 15；同时存在逗号/小数点时规则不同。非零小数在 .NET 整数解析中抛 OverflowException，不能按 FormatException 回退；前置分隔符或小数部分再次出现分隔符则按格式失败保留原 Ep。全部保留真实红报告与相应用例，未修改预期来接受错误输出。

比较协议新增严格 `expectedErrors`：默认仍拒绝任何执行错误；负向场景必须声明双方完整错误，改变错误、意外成功或额外请求/结果/种子仍失败。只在错误精确匹配后将缺省 results 视为空列表。比较器补齐错误预期/过期预期回归，12 项通过。引擎离线回归证明非法季、数值溢出及非整数在搜索 HTTP 前拒绝；此断言不扩大为整个账号认证层从未发请求。

最终 `/tmp/moji-indexer-diff-structured-final.json`：21/21 PASS，含 20 个正常流程与 1 个预期查询拒绝，0 未解释差异，原有 3 个下载行为差异单列。完整 Go、cardigann/indexer/defs/replay/cmd 完整 race 与相关 vet 通过；固定 v11 579/579 静态编译、1 个已有定义警告、0 失败（`/tmp/moji-v11-query-audit.json`）。没有真实站点、任务、数据库或部署操作。

## 人工文字与图片验证码差分

新增 captcha-text、captcha-image，沿用同一 YAML 与响应执行落地页→人工挑战→原表单续提交→登录校验→搜索→取种。Jackett runner 调用固定源码的 GetConfigurationForSetup(false)，将答案写入实际动态配置，再调用 ApplyConfiguration；Moji 调用 Authenticate/AnswerChallenge，不另开执行实例。参考程序变动后在全新 `/tmp/moji-indexer-diff-reference-2` 重建并核验 31 个 DLL 与 runner 摘要，未绕过 provenance。

首次实际报告 `/tmp/moji-indexer-diff-captcha-first.json` 两个流程均完成，但暴露登录落地页缺少站点 Referer、验证码 GET 误带表单 Content-Type。现已修正：落地请求设置站点来源；验证码 GET 使用加入 POST Content-Type 前的独立头快照，保留规则自定义头及登录页 Referer。Cookie、原始隐藏字段、答案表单字节与单次落地/单次提交有离线回归，缺答案中止、不带挑战的答案拒绝且不搜索。

最终 `/tmp/moji-indexer-diff-captcha-final.json`：23/23 PASS（22 正常、1 预期查询拒绝），0 未解释差异，原有 3 个下载行为差异保持单列；两个人工挑战场景不需例外。比较器显式比较挑战类型/文字/图片摘要，新增展示变化与答案边界校验，共 14 项 Python 测试通过。完整 Go、cardigann/indexer/login/check/replay 完整 race、相关 vet、文档链接与 diff 检查通过。图片 MIME 不经 Jackett 公开配置保留，故不在差分认证范围；合成图片仅检验字节传递和流程，未认证浏览器图片识别。未操作真实数据、站点、任务或部署。

后续新增 captcha-text-rejected：提交后匹配规则 error selector，两边都只发登录页 GET/表单 POST，不继续登录测试、搜索或取种。Moji 既有脱敏错误与 Jackett 原站点文字不同，使用严格 expectedErrors 分别固定完整消息，不采用宽泛忽略；没有生产代码改动。最终全组 `/tmp/moji-indexer-diff-captcha-rejection-final.json`：24/24 PASS（22 正常、2 预期拒绝），0 未解释/3 原有下载行为差异，14 项比较器测试通过。

## 可空分页、数字查询字段与声明参数名

核对固定 Prowlarr [请求生成器](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannRequestGenerator.cs)、[SearchCriteriaBase](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/IndexerSearch/Definitions/SearchCriteriaBase.cs)和 [HttpRequestBuilder](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Common/Http/HttpRequestBuilder.cs)，并读取固定 Jackett TorznabQuery、StringUtil、WebClient 及 BaseIndexer。引擎 Query.Limit/Offset 改为可空整数，缺省不生成模板文本，显式 0 保留字符串 "0" 的真值；生产 indexer.Query/API 不变，服务仍规范化为 limit=100/offset=0，BuiltinAdapter 显式传入，实际会话/取种链路回归证明未丢失。

普通声明 input 名称现在按 Prowlarr 对 GET/POST 都原样保留，如 filter[]；只编码值。固定库新增 search.literal-input-name 特征，122 条搜索路径、120 个定义包含此类括号名称；无对应语义的继承关闭路径不计入。这里证明字节行为改变，不据此推断这些站点原先不可用。GET 与 Jackett 一致；POST 则与其 FormUrlEncodedContent 编码名称行为不同，按现代规则解释器实现选择记录精确正文差异。

初始新 fixture 使用了 schema 不允许的特殊字符，已改为合法括号名；没有放宽 schema 迎合测试。正 offset 用例还发现 Jackett BaseIndexer 在 Cardigann 不支持分页时直接跳过请求，因此真实差分使用 offset=0；正 offset=40 的渲染由离线请求测试及 Prowlarr 源码证明，不伪称已跑 Jackett 解释器。

query-ids 的真实红报告 `/tmp/moji-indexer-diff-query-ids-first.json` 证明年份/数字 ID 原先只透传：002026/00012 与上游 2026/12 不同。现对 Year、TVDBID、TVRageID、TMDBID、TVMazeID、TraktID、DoubanID 做 Int32 规范化与边界拒绝；缺字段不填值，明确空字符串为错误。初版误将空串当作未指定，真实 JSON.NET 输入对照纠正了这一点；报告 `/tmp/moji-indexer-diff-id-boundaries-first.json` 留证，未通过例外接受多发请求。非法值在引擎搜索 HTTP 前拒绝，既有账号认证层可能已执行；ErrInvalidQuery 保留稳定映射 INDEXER_QUERY_INVALID，实际内置适配器回归证明不发搜索请求。

新增 query-keys-get/post、query-pagination-unset/zero/values、query-ids/empty、query-id-overflow-error 共 8 个实际场景。参考 runner 只补 Torznab TVRageID→RageID 名称映射，在新 `/tmp/moji-indexer-diff-reference-3` 重建并核验 31 个 DLL/源码摘要。最终 `/tmp/moji-indexer-diff-query-fields-verified.json`：32/32 PASS（28 正常、4 预期拒绝），0 未解释、5 已解释差异（原有下载 3 项及现代 Prowlarr 查询契约 2 项），14 项比较器测试通过。完整 Go、cardigann/indexer/defs/check/两个 CLI/cmd 完整 race、相关 vet、文档/diff 检查通过。固定 v11 579/579 静态编译、1 个已有定义警告、0 失败，报告 `/tmp/moji-v11-query-fields-audit.json`。未操作真实配置/数据库/站点/任务/下载器，也未提交或部署。

## Magnet 获取、身份与成功缓存

新增 magnet-download/field/redirect/page/before，分别从 download/magnet 结果字段、HTTP Location、HTML 下载选择器及 before 响应获取同一完整 magnet。Jackett BaseIndexer.Download 把中间/跳转 magnet 作为 UTF-8 bytes 返回；runner 现在仅将精确 magnet: 前缀的字节切到报告 magnet 分支，其他内容继续比较种子 SHA-256，不做 URI 重排或 tracker 删除。参考程序在新 `/tmp/moji-indexer-diff-reference-4` 重建，31 个 DLL 和源码摘要核验通过。

五个真实解释器用例首轮全部通过（`/tmp/moji-indexer-diff-magnets-first.json`），没有为其新增例外。15 项比较器测试证明 tracker 内容改变会失败，并固定这些场景必须存在。引擎纯解释流程不触发 downloader；原有任务测试另外覆盖已接受 magnet 的先落盘、原始链接提交与脱敏公开投影，不能将解释器对照扩大为真实下载器验收。

进一步实际回归复现 ArtifactCache 只检查 xt 非空，短哈希、错误 namespace 和无效 base32 都能当成功产物缓存。现由纯 torrentmeta.ParseMagnet 给 Cardigann 和统一 Artifact 校验共用，64 KiB 预算；验证 btih hex/base32 并形成规范哈希，完整 URI 参数保留，成功缓存仍按完整原始内容隔离。同一测试还发现命中缓存丢失内部已解析哈希，现从 blob 一并恢复。缓存负向回归证明无效资源不缓存、不自动重试；正向回归证明相同 v1 哈希不同 tracker 不互换、命中仅获取一次。

采用 [BEP 9 magnet 格式](https://www.bittorrent.org/beps/bep_0009.html)及 [BEP 52](https://www.bittorrent.org/beps/bep_0052.html)确认 v2 的 btmh 十六进制 multihash，保留合法 0x12/0x20 SHA-256 topic，避免统一校验把原外部入口可接受的有效 v2 链接缩为 v1。混合 URI 保留两个 topic，缓存内部身份优先 v1；这里只证明 URI/缓存语义，不表示已验证完整 v2 种子解析/任务下载，未接触真实下载器。自动由 infohash/title 生成 magnet 时的上游公共 tracker 默认值仍须另行对照。

最终 `/tmp/moji-indexer-diff-magnets-verified.json`：37/37 PASS（33 正常、4 预期拒绝），0 未解释/5 原有已解释差异；完整 Go、torrentmeta/cardigann/indexer/downloads/taskruntime/replay/cmd 完整 race、相关 vet、15 项 Python 比较器、文档/diff 检查通过。未修改 UI 或重复相关 Web 验证；未操作实际配置、数据库、站点、任务或下载器，未提交部署。

## schema 矩阵与全部过滤器管线的预编译

新增 [A–F/schema 证据矩阵](2026-11-03-indexer-acceptance.md)，源码核对当前 22 个 schema definitions、120 个具名 properties；逐项字段覆盖比对无遗漏，71 个类别枚举都有 ID，25 个过滤器名称都有执行分支。这只证明审查映射完整，不证明每个分支语义已认证；矩阵明确区分宿主元数据、局部回归、待判上游适用域和已观察缺口。README/部署示例/smoke 的 Jackett 必需假设仍是未完成交付，未通过改设计排除。

矩阵发现 validateSyntax 仅预编译名为 filters 的列表。回归先复现 keywordsfilters/preprocessingfilters 的非法正则/参数数量通过 Load、特征统计漏项；现统一预编译/计数，安装更新使用同一入口。实际临时归档 UPDATE 回归确认坏管线不发布且当前库仍可用。新增检查首轮全库 578/579，原因是 thepiratebay 关键词管线用到未适配的 .NET IsCJKUnifiedIdeographs；未通过放宽检查或排除定义消除失败。

依据 [dotnet/runtime v9.0.0 RegexCharClass](https://github.com/dotnet/runtime/blob/v9.0.0/src/libraries/System.Text.RegularExpressions/src/System/Text/RegularExpressions/RegexCharClass.cs) 的块边界 U+4E00..U+9FFF 修正兼容层，保留已有 Cyrillic 处理；BMP 正/反字符类、边界、字面量转义与未知属性拒绝有回归。当前只适配这两个实际遇到的 block，全部 Unicode/UTF-16 语义仍未认证。query-cjk-keywords 使用同一规则/HTTP 响应执行真实 Jackett 对照，无新增例外。

最终固定 579/579 静态编译、1 个已有定义警告、0 失败，报告 `/tmp/moji-v11-all-pipelines-verified.json`；实际差分 `/tmp/moji-indexer-diff-all-pipelines-verified.json`：38/38 PASS（34 正常、4 预期拒绝），0 未解释/5 原有已解释差异。完整 Go、cardigann/indexer/defs/check/replay/cmd 完整 race、相关 vet、15 项比较器与文档/diff 检查通过。没有 UI 改动、实际配置/数据库/站点/任务/下载器操作或提交部署。

## simpleCaptcha 与 andmatch 声明式语义

矩阵初审发现的 simpleCaptcha 缺少 submitme=X 已由真实红报告复现；现在自动获取 hash 后按原表单位置覆盖 captchaSelection/submitme，保留隐藏字段、Cookie 和声明编码。新增 simple-captcha 实际完成登录→搜索→取种；独立回归覆盖空 images 和取消后不提交、不重试，验证码 GET 不带 POST Content-Type。

andmatch 原实现没有字符上限，使用原关键词且没有停用词/ID 能力旁路。新增十个合成场景，覆盖正上限、UTF-16 截断、单字 CJK、and/the/an、处理后的 .Keywords、空处理结果的清洗回退、缺词排除、IMDb 和 allowtvsearchimdb。按照固定 [Jackett ParseRowFilters](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Indexers/Definitions/CardigannIndexer.cs) 与 [TorznabQuery.MatchQueryStringAND](https://github.com/Jackett/Jackett/blob/cfdc645711c487d05cb55932e394b2370154d81e/src/Jackett.Common/Models/TorznabQuery.cs) 修正；参数必须是 Int32，错误在规则安装预编译时带源位置拒绝。运行时异常产生 ROW_FILTER_FAILED issue，不能当完整空结果缓存。

明确兼容选择：固定 [Prowlarr Cardigann](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/Cardigann.cs) 将 andmatch 交给 [IndexerBase.FilterReleasesByQuery](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/IndexerBase.cs)，忽略字符上限，停用词多出 of，标题/描述至少匹配两个词即可，且宿主决定 ID/RSS 跳过。Moji 当前选择 Jackett 的声明式逐词规则与显式参数，符合既定相同场景比较目标；这不表示两个宿主本身同义。完整 Unicode OrdinalIgnoreCase 和重复 andmatch 的首次词缓存边界尚未认证。

实际红报告 `/tmp/moji-indexer-diff-row-filters-first.json` 保留原差异，修正后新增 11 例无例外通过。最终 `/tmp/moji-indexer-diff-row-filters-final.json`：49/49 PASS（45 正常流程、4 精确预期拒绝），0 未解释/5 原有已解释差异。固定语料 `/tmp/moji-v11-row-filters-audit.json`：579/579 编译、1 个既有警告、0 失败。完整 Go 首轮五包因沙箱禁止 httptest 监听失败，允许临时本地监听后受影响包全部通过，其余包首轮通过；cardigann/defs race 通过，indexer/taskruntime race 同样在允许临时监听后通过；相关 vet、15 项比较器通过。没有 UI 改动或实际配置/数据库/站点/任务/下载器操作；前一阶段 amd64 镜像早于本次引擎修正，不能当作当前源码镜像验证。未提交部署。

## 行计数的 JSON 适用域

核对同一固定 Jackett CardigannIndexer 及 [Prowlarr CardigannParser](https://github.com/Prowlarr/Prowlarr/blob/3326e2d4aa978f35fa7e44f66758ebfc8f553d67/src/NzbDrone.Core/Indexers/Definitions/Cardigann/CardigannParser.cs)：rows.count 仅 JSON 分支使用，在取 rows 前按 Int32 尝试解析；小于 1 则为空，缺失/异常/非法计数不阻止正常取行。HTML/XML 的 after/dateheaders 与 JSON 的 attribute/multiple 分工也不能因 schema 共用 SelectorBlock 就泛化；固定源码的行选择没有消费 case/remove/text 字段。

六个实际合成场景先证明 Moji 缺失 JSON count、HTML 却额外执行 count，红报告 `/tmp/moji-indexer-diff-count-first.json` 保留三处失败。首轮 HTML fixture 缺少 search.html 导致工具报错，补齐合成文件后重新执行完整六例，没有用缺 fixture 的运行当兼容证据。

parse.go 现于 JSON 行选择前执行计数；零/负数提前结束，缺失/非法/越界继续正常解析，非负有效值保留为可选 ReportedCount；取消优先返回，不被辅助计数的宽松策略吞掉。移除 HTML 路径多余 count 执行。回归覆盖有符号/空白、Int32 上下溢、小数、过滤后零值与缺行提前结束，没有新增 API/配置或任务持久字段。

最终 `/tmp/moji-indexer-diff-count-final.json`：55/55 PASS（51 正常、4 精确预期拒绝），0 未解释/5 原有已解释差异，六个新增场景无例外。完整 Go 在允许临时 localhost httptest 的隔离环境通过；cardigann/indexer/defs 完整 race、相关 vet 与 15 项比较器通过。本次未改变 Load/schema/预编译入口，固定语料编译结果沿用上一节实际运行的 579/579，不伪称重新执行。未重复无关 UI 检查；没有真实配置/DB/站点/任务/下载器操作或提交部署。

## 参考构建的 NuGet 依赖锁定

依据 [NuGet 官方锁定模式](https://learn.microsoft.com/en-us/nuget/consume-packages/package-references-in-project-files#locking-dependencies)，用固定 SDK 在独立临时源码副本、既有包缓存和断网容器中生成三个 packages.lock.json，覆盖 runner、Jackett.Common、DateTimeRoutines；执行入口闭包为 36 个包，逐一版本比对与此前通过对照的参考程序完全相同。锁文件只含 net9.0 的解析版本、依赖边和内容 hash，不是新增 Moji 运行时依赖。

锁快照位于 scripts/indexer-diff/nuget-locks，SHA-256 纳入 upstream.lock.json；构建前验证完整固定目标集合及来源 hash，复制进临时源码后只能 --locked-mode restore，再验证锁文件未变。编译 --no-restore 使用另一断网容器。provenance 和比较入口同时验证源码锁、已安装锁、固定 pin、runner 与 DLL hash，旧无锁 reference-4 已实际被拒绝，不能继续拿旧构建冒充新参考。

全新 `/tmp/moji-indexer-diff-reference-5` 空包缓存实际恢复/编译成功，31 个 DLL 摘要留证，日志 `/tmp/moji-indexer-reference-locked-build.log`。独立临时副本将 AngleSharp 要求 1.8.6 改为 1.8.5，真实 SDK --locked-mode 以 NU1004 拒绝，三个锁文件均未改写，日志 `/tmp/moji-indexer-reference-lock-drift.log`；随后恢复临时源文件，没有放开 locked-mode 回退。五项 Python 回归覆盖源锁/已安装锁缺失或变化、未知路径、缺目标拒绝，主 CI 和兼容 CI 都已接入。

新参考执行 `/tmp/moji-indexer-diff-locked-final.json`：55/55 PASS（51 正常、4 预期拒绝），0 未解释/5 原有已解释差异；5 项构建锁测试、15 项比较器、版本/发布脚本各 12 项及文档/diff 检查通过。版本/发布脚本为离线脚本测试，不是 ARM 镜像构建证据。本阶段没有 Go/UI/公开契约改动，不重复已通过的 Go/Web suite；也不承诺上游带 BuildDate 的 DLL 可跨次构建字节相同。没有真实配置/数据库/站点/任务/下载器操作或提交部署，远端 CI 未运行。

## 获取取消、magnet 编码与待确认的公开 tracker 默认

新增回归实际复现：Fetch 开始前已取消，直接 magnet/仅 hash 路径仍返回成功；infohash 选择器将 context.Canceled 改写为通用匹配错误。现 Fetch 入口先检查取消，hash/title 选择器用 %w 保留 cause，普通下载 fallback 也不吞取消。已开始的 HTTP 如果确实收到完整种子，仍保留已确认资源供外层成功缓存/任务 generation fence 使用，不能因本次取消就丢弃已消耗抓取的回执。测试覆盖入口零请求、选择器取消不继续/不重试、已确认种子在取消边界不丢失。

新增 infohash-page/infohash-before 两条实际对照，红报告 `/tmp/moji-indexer-diff-infohash-first.json`：双方的请求/结果相同，最终 URI 在标题安全字符和 Jackett 默认公开 tracker 列表处不同。标题编码现统一 generatedMagnetURI，UTF-8 WebUtility 安全字符 !*() 与上游一致；三个生成入口共用，hash/title 作为完整参数值编码，不能从 hash 字段插入额外参数。单独回归确认 Unicode 名称及参数注入拒绝。

公开 tracker 是否沿用既有 Jackett 行为属于下载默认值，已按项目指引向用户发出选择，尚无回复，不能将超时当同意。两条原始 fixture 保留于 scripts/indexer-diff/pending-cases，可显式 --case 运行，无已接受差异例外，未计入 55 个通过数；此项仍必须完成后再放回正式 cases，没有通过移出路径把兼容目标改小。没有先附加公开 tracker 或以比较器忽略它们。

当前已接受语料 `/tmp/moji-indexer-diff-cancel-encoding-final.json`：55/55 PASS，0 未解释/5 原有已解释差异。完整 Go、cardigann/indexer/taskruntime race、相关 vet、5 项锁校验/15 项比较器通过；引擎 benchmark 另外以 race 运行全部六项各 1 次通过。固定 schema/加载入口未改，语料静态编译沿用前次证据；没有真实站点/服务/配置/数据库/下载器操作或提交部署。tracker 选择不会阻止其他独立兼容/性能工作继续。

## after 分组与 dateheaders 的匹配范围

新增 rows-after/dateheader-sibling/parent/missing-optional/missing-required/invalid-nearest/nested-scope 七条实际场景。首轮六例中完整分组及普通日期查找均一致，invalid-nearest 复现 Moji 在最近匹配日期无效时继续扫描旧表头；独立 nested-scope 复现任意向上遍历祖先导致借用页面其他区域日期，报告 `/tmp/moji-indexer-diff-dateheaders-first.json` 与 `/tmp/moji-indexer-diff-dateheader-scope-first.json`。

固定 Jackett CardigannIndexer 与此前读取的同 pin Prowlarr CardigannParser 都在首次 selector 匹配后退出查找，再解析日期；查找没有前兄弟时仅尝试父节点的前兄弟，不能跨空的多层父节点继续上溯。parse.go 已按此修正。无效首次日期产生 FIELD_VALUE_INVALID 并拒绝该行，即使 optional 也不借用旧日期；必需日期缺失仍为 REQUIRED_FIELD_MISSING。这些 issue 由内置适配器标记 Partial，不能把异常行被排除后的结果当作完整空集缓存。

新增本地回归覆盖兄弟/单层父兄弟、跨层拒绝、无效最近日期（含 optional）、可选缺失与诊断。测试初版显式 optional:false 被固定 v11 schema 拒绝；改为省略该属性来表达必需，未放宽 schema。七条正式场景无新增例外，最终 `/tmp/moji-indexer-diff-dateheaders-final.json`：62/62 PASS（58 正常、4 精确预期拒绝），0 未解释/5 原有已解释差异。完整 Go、cardigann/indexer/taskruntime race、相关 vet、15 项比较器、文档/diff 检查通过。两条 tracker 待确认场景仍在 pending-cases，不计通过。

未改 schema/Load，不重复固定 579 定义静态编译；未修改 API/UI 或重新运行无关浏览器/性能矩阵。完整 after 分组已实际对照，但不足组、全部 XML/DOM/日期格式组合和相对时钟不能据此判全部覆盖。没有真实服务/配置/数据库/站点/下载器操作或提交部署。

## 查询派生标记、IMDb 短 ID 与设置值

固定 Jackett TorznabQuery 和 CardigannIndexer 装填 15 个 .Query.Is* 标记；Moji 原先均缺失。新增九个实际请求场景复现模式、ID/RSS 判断缺项，包含 Year=0、没有 Season 的原始 Ep、明确空 IMDb 和空 Genre：空 IMDb 仍是 ID 查询；空 Genre 的 IsGenreQuery 为真但 IsIdSearch 为假。applyDerivedQueryFields 现根据原始参数存在性、非空内容和规范化正季数派生，不能从派生 Episode 非空反推 IsIdSearch；附本地负季/空白 episode/Publisher 回归。

三个 IMDb 场景证明上游短 ID 是 TrimStart('t')，不是删除一次 tt；修正为去除任意数量的起始小写 t，原始 IMDBID/大写 TT 保留，不冒充 IMDb 身份格式验证。红报告 `/tmp/moji-indexer-diff-query-derived-first.json`：九个 flag 和两个短前缀失败，大写控制通过；没有接受请求差异掩盖问题。

设置变量另核对固定 Prowlarr CardigannBase 与本地 Jackett：Jackett checkbox 从 .True 取 "True"，Prowlarr 源码用字面 ".True"。Moji 明确按既定 Jackett 执行对照修正为 "True"，使值比较和原文输出一致，非空布尔 if 行为保持；三个实际场景覆盖规则默认 true、显式 false、显式 true。没有将该分支描述为两个宿主本来同义。年份未改：Moji .Today.Year 沿用 Prowlarr 当前年份；Jackett 一月份取上一年；这里只是固定源码差异，未伪称实际固定参考时钟已覆盖一月。

用生产 Load 核对固定归档全部 579 定义（SHA-256 再核对与 pin 一致），914 个 select 设置都有非空默认；两个额外场景证明默认 b 和用户覆盖 a 不受 options 声明/排序次序影响。临时独立 YAML 扫描不支持社区转义，已改为生产加载器核对并清理临时源码目录；该检查实际 Load 全部通过，但没有收集新的 warning 明细，不替代原语料工具的特征报告。schema 允许但当前库没有的空 select 默认 fallback 仍属边界，未因库中未出现而删除该能力目标。

首轮完整差分在布尔 JSON settings fixture 处被 Go CLI 字符串边界拒绝，改成与实际引擎适配一致的 "true"/"false"。比较器新增前置类型检查及回归，防止 malformed fixture 到执行阶段才失败；没有改 C# 参考、生产 typed patch 或接受错误例外。最终 `/tmp/moji-indexer-diff-query-derived-final.json` 77/77，加 `/tmp/moji-indexer-diff-select-default-final.json` 2/2，同一生产/参考版本累计 79 个正式场景全通过（75 正常、4 预期拒绝），0 未解释/5 原有已解释差异；明确不是一条命令跑 79 例。

完整 Go、cardigann/indexer/admin/taskruntime race、相关 vet、16 项比较器与文档/diff 通过。两条 tracker 待选择仍不计通过；没有 UI/公开 schema 改动、实际配置/数据库/站点/下载器操作或提交部署。后续命名参数资格、其他默认值、相对时钟和 .NET/filter 边界继续在 R-004，不能用本节代表所有变量已经认证。

## 用户确认公开 tracker 默认与 infohash 正式验收

2026-11-03 用户明确选择“沿用 Jackett，附加公开 tracker”，已落实到共用 generatedMagnetURI：结果的公开/半私有 hash fallback、原页和 before 页 infohash/title 使用固定 Jackett MagnetUtil 的十个默认地址，参数顺序/UTF-8 编码一致。站点原始 magnet 不追加/替换 tracker，私有 hash-only 结果不自动生成公开 fallback；单独 Fetch 回归验证无额外 HTTP。规则下载与配置结构未变，没有新增自动联网拉 tracker 列表。

两个待决场景移回 cases，pending-cases 目录移除，新增 public/semi-private/private 三个 hash-only 场景，私有场景比较搜索投影（不调用无资源的 reference 抓取分支），本地回归另外证明 Fetch 拒绝公开 fallback。这里不将只比较搜索投影说成已验证私有站点实际下载。

初次完整 84 例报告 `/tmp/moji-indexer-diff-public-trackers-final.json` 为 82 通过：两个中文 hash-only case 共四处差异来自参考 runner 的 Uri.ToString()，并非 tracker 列表不一致。固定 Jackett TrackerCacheResult 保留 typed Uri；独立固定 SDK/Newtonsoft.Json probe（`/tmp/moji-uri-projection/build.log`）证明 JSON 序列化保留原始百分号编码，而 ToString 为显示用途会展开中文。runner 改用实际 JsonConvert 序列化 roundtrip，仅修正报告投影，没有改 Jackett 解释类、Moji 原始 URI 或 Python 比较规范。17 项比较器仍确认编码/显示字符串差异会报错，未通过规范化或接受例外消除差异；不冒称对实际 Jackett Server 做了在线 API 测试。

参考程序在全新 `/tmp/moji-indexer-diff-reference-6` locked-mode 构建并核验 31 DLL/3 个包锁；旧 reference-5 因 runner 摘要变化被实际拒绝。最终 `/tmp/moji-indexer-diff-public-trackers-verified.json`：84/84 PASS（80 正常、4 精确预期拒绝），0 未解释/5 原有已解释差异，五个新增/恢复场景没有例外。

完整 Go、cardigann/indexer/torrentmeta/taskruntime race、相关 vet、17 项比较器/5 项锁校验与文档/diff 通过。早先局部 Go 的 indexer 包因沙箱禁止 httptest 监听失败，完整回归在允许临时监听后通过，未记为产品失败或未经执行就记通过。本次未改 schema/规则 Load/可见 UI，不重复静态语料、浏览器或旧性能基准；没有真实服务/配置/数据库/站点/tracker/下载器操作或提交部署。用户确认项已关闭，其他完整验收继续。

## 命名查询参数的资格与持久接受边界

新回归实际复现：参数 map 的 Q/Type/Limit 等可以覆盖 typed 查询上下文，小写 year 被引擎忽略；统一服务只检查字节长度，未知/重复大小写字段仍读来源；Jackett JSON 搜索忽略额外参数和 offset。失败回归没有依靠真实站点，均从入口到假 HTTP 或来源快照计数观察。

新增窄共享包 internal/indexerquery，由统一入口及独立解释器两个真实消费者复用；18 个可写字段来自已支持的结构化查询组，大小写不敏感，保留值/空串并返回独立 map。未知/保留字段、重复大小写别名、超限输入都拒绝，不能用参数覆盖模式/关键词/分页/类别/派生变量；Engine 映射 ErrInvalidQuery，统一入口为 INDEXER_QUERY_INVALID。没有新增通用 provider、状态层或重复查询模型。实际托管 runtime DB 回归确认非法参数不创建 run，合法 year 以 Year 身份保留原值后持久接受。

外部 Jackett 现有 JSON 适配仅表达关键词/类别，结构化参数或非零 offset 现在在任何 HTTP 前返回 INDEXER_CAPABILITY_UNSUPPORTED；已有电影等非 search 模式拒绝保留，没有谎称新增了外部 Torznab 结构化搜索支持。现有 UI 仍是关键词/分类，API 文档说明资格；不支持提示从“检查规则版本”改为调整条件/更换来源，中英及内置 fallback 同步。

新增 query-parameter-case，以 year/imdbid 小写同一条件执行真实 Jackett 对照。最终 `/tmp/moji-indexer-diff-query-contract-final.json`：85/85 PASS（81 正常、4 预期拒绝），0 未解释/5 原有已解释差异。完整 Go、cardigann/indexer/torrentsearch/graphqlapi race、相关 vet、17 项比较器通过。make generate-moji-graphql 已完成后端和 Web 绑定生成，Web 生产构建及四文件/10 项 i18n 检查通过；初轮 Web/Vitest 因沙箱 spawnSync sh EPERM 失败，允许执行本地版本脚本后通过，不把环境失败记作通过。

未改可见布局或新增表单，没有重复浏览器视觉矩阵；本轮文案依据实际不支持路径，调整不会要求用户无效更新规则。未改规则 schema/Load，沿用此前静态语料证据；没有真实配置/数据库/站点/下载器操作或提交部署。其他完整验收继续，不能将入口约束当作全部 v11 行为的完成证明。

## 仍需继续的差异审计

1. 可空分页、声明参数名、数字 ID、15 个 flag、IMDb 短前缀、checkbox/select 默认/覆盖及命名参数资格已有上述证据；空 select 默认 fallback、其他默认/相对时钟仍需结合完整 schema 能力矩阵核对，不能以代表性查询证明全部语义。
2. 已有真实上游程序 85 个正式场景与兼容 CI 配置，参考依赖锁定、新构建对照及用户确认的 infohash/tracker 默认均已通过。其他错误/取消边界、相对时钟策略、剩余 .NET/日期/filter 行为、来源/许可及显式开启的真实站点和代理/solver 验收仍未完成。源码依据、可执行语义覆盖与实际站点证据分开报告。

<a id="ci-go-coverage"></a>
## CI 默认离线覆盖

R-004 的新增包必须实际运行自身测试。`ci.yml` 的 Go job 已从 graphqlapi/taskruntime/cmd 三包改为 `go test ./...`，两个 Python 脚本检查保留，覆盖存储与全部新索引模块；G-001 的配置缺口关闭。

本地以 `go test -count=1 ./...` 完成一次全包无测试结果缓存运行，两个 Python 脚本各 12 项通过。没有新增外部服务要求；真实服务测试仍显式开启。本轮 Cardigann、indexer、definitions、CLI、cmd 的完整 race/vet 已通过；随后 download.before 修正另经 Cardigann 与 indexer race/vet 和全量 Go 检查。工作流未提交/推送，没有声称远端 Actions 已运行。语料与上游程序差分 CI 是 R-004 后续工作。

本轮测试只用临时数据、假 HTTP 和固定离线归档；没有操作真实配置、数据库、站点、下载或部署。
