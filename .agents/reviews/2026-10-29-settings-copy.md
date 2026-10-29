# 设置文案审核（2026-10-29）

范围：七个设置分区的标题、标签、选项、提示、空态、状态及确认文案；同时审阅中英文资源。原则：能从控件直接理解的功能不加说明；说明用于范围、非直观行为或后果。新增界面和已保留的字段统一遵守此原则。

| 分区 | 核对源码 | 文案处理 |
| --- | --- | --- |
| 连接 | pkg/stash/client.go、pkg/jackett/client.go、pkg/qbittorrent/client.go；ServiceConnectionSettings | 删除服务启用/未启用、重复服务名和填写/保存说明；保留地址、凭据及真实测试结果。管理密码与 API key 区分。 |
| 索引站点顺序 | internal/tracker/jackett_service.go；internal/taskruntime/candidate_selector.go | 保留自动逐站回退与手动查询继续收集后续站点结果的区别，不枚举全部网络错误。 |
| 下载器 | internal/downloader/service.go | 说明只清理关联任务全部完成的种子；入库后清理实际在下次同步，不承诺即时。文件删除保留独立副本/共用源文件限制；速度全局生效、0 不限速。 |
| 媒体交付 | internal/taskruntime/stash_scan.go；internal/mediafiles；internal/stashsync/ingest.go | 删除每个根目录的实现细节；成对路径只说明同一目录的两个视角。保留移动删除源文件的后果、转码成本、字幕在合并/单文件中的差异。 |
| 来源与缓存 | internal/metadata；internal/stashboxcache；internal/subscription/service.go | 不把优先级说成来源开关；不把浏览缓存说成订阅真值。去掉 SQLite、快照数等无关实现说明，保留失效后刷新及浏览的失败回退。 |
| 任务与订阅 | cmd/moji/main.go；internal/subscription/release_policy.go | 删除运行启用状态摘要与开关重复说明；移除“首次全量回查历史”的不成立描述。保留关闭任务处理的影响、检查间隔负数/0 语义、发行策略范围和 VR 严格叠加。 |
| 资源选择 | internal/taskruntime/candidate_selector.go | 删除重复摘要、无参数的标题相似度说明；保留硬条件和排序的区别、正则目标转小写、种子检查限制及单视频为软偏好。枚举改为用户可读文案，内部值不变。 |
| 删除策略 | internal/taskruntime；internal/downloader | 区分手动删除与自动清理；保留源文件删除的选项，不把二者描述成同一个策略。 |
| 系统 | internal/imagecache/service.go；web/src/hooks/useInterfaceSettings.ts | 图片缓存关闭后既有图片仍参与清理，容量和保留期继续可编辑。语言解释实例设置与浏览器偏好的区别；开发者模式仅说明新增的性能入口。清空确认保留数量、空间和重新下载后果。 |
| 日志、关于 | LogsSettingsTab、LogEventStream、AboutSettingsTab | 删除重复筛选值、就绪、来源、空态说明和版本标签；区分复制当前列表与下载日志文件。 |

空闲和成功保存不显示固定说明；保存中和失败反馈保留。错误/测试结果为真实请求反馈，不能因为精简说明而删掉。没有改变任务、发行、清理或缓存的领域语义。
