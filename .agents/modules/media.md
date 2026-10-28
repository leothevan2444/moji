# 文件选择、媒体交付与 Stash 入库

核对日期：2026-10-27；代码基线：`86f69e5`；状态：当前实现。

主要源码：[文件计划](../../internal/mediafiles/plan.go)、[产物准备](../../internal/mediafiles/prepare.go)、[runtime 文件控制](../../internal/taskruntime/media_files.go)、[恢复](../../internal/taskruntime/restart.go)、[入库推进](../../internal/taskruntime/stash_ingest.go)、[Stash 集成](../../internal/stashsync/ingest.go)。用户行为说明见 [媒体交付](../../docs/media-delivery.md)。

## 领域边界

Moji 决定下载哪些文件、整理产物和路径，并编排后续工作。Stash 负责 scan、generate、identify 与媒体库元数据。Moji 不直接替代 Stash 写影片元数据，也不生成或翻译字幕。

`SCANNING` 不等于一次扫描请求已成功；完整入库还包括生成、识别和结果验证。上游 job ID 与阶段检查点持久保存，恢复应核对已有执行结果，避免重复扫描或重复交付。

## 下载文件计划

文件分为 `MAIN/SUBTITLE/DISCARD/UNKNOWN_VIDEO`，依托可信番号识别，保留 qB 原始 index、相对路径、大小、理由、部件顺序、字幕选择和优先级应用状态。广告、样片、截图等排除；不确定视频或顺序必须等待人工选择。

qB 使用 `stopCondition=MetadataReceived`；非 magnet 也先停止添加。得到元数据后暂停，持久化选择意图并应用原始 index 的优先级，再启动所选下载。不能按排序后的 UI 下标改 qB 文件；下载完成按所选文件进度判断。

`resolveTaskFiles` 只接受服务器已有计划中的合法视频 indices，并按播放顺序提交。手动选择、字幕开关和“尚未应用”的状态在外部优先级/启停操作前持久化；失败或重启能够继续核对。

## 产物与安全发布

四个路径根必须明确配置：qB 下载根、Moji 所见下载根、Moji 库根、Stash 所见库根。路径文字相同不保证共享同一实际目录；路径映射必须验证包含关系。

| 输入 | 交付产物 | 媒体工具 |
| --- | --- | --- |
| 一份独立影片，或去重后仅一份 | `CODE/CODE.ext`，保留原扩展名与字节 | 不调用 FFmpeg/ffprobe |
| 多份兼容影片部件 | `CODE/CODE.mkv`，无重编码拼接 | FFmpeg 和 ffprobe |
| 部件不兼容、顺序未知、字幕冲突 | 阻塞等待人工处理 | 不静默转码或猜顺序 |

准备任务以 `TRANSFERRING/PENDING` 排队，最多一个任务实际处理。工具执行使用应用生命周期 context，关闭时取消并留可恢复检查点；删除任务会取消对应准备。

验证源文件大小、包含关系和 hash，去除字节完全相同的视频。工作先写到带 `.stashignore` 的私有临时目录；全部验证完成后在同文件系统 rename 发布。无关或被修改的既有目标不能覆盖或冒领。

`.moji-receipts` 在提交给 Stash 的影片目录之外保存产物 hash 和任务所有权，帮助识别已完成交付。回执是恢复范围的一部分，不能随意清理。

COPY 保留源；MOVE 先停 qB，完整发布后才删除选中的源，恢复允许部分源已经删除。自动删种还要遵守独立副本和交叉做种保护，见 [运维](operations.md)。

## 字幕和生成资源

外置字幕保留默认关闭，开关覆盖所有语言，不做语言猜测；明确属于其他番号的字幕排除。单文件交付保持原影片字节，因此其嵌入轨道不会因开关被改写。

多部件嵌入字幕遵循保留设置。支持的文本部件字幕转为 SRT 并按前置视频时长平移；整片字幕保留原格式。位图部件字幕或冲突不能假装已正确处理。

Stash 全视频 transcode 由 `ingest.generate_transcodes` 控制，默认 `false`；封面、预览、sprite 与 pHash 仍由已有生成流程安排。不要因“生成播放资源”自动开启昂贵全视频转码。

## 修改时验证

- 单文件原格式与字节保持、COPY/MOVE、回执恢复、越界路径和已修改目标拒绝。
- 原始 qB index、人工顺序、意图先落盘、所选进度、重复操作和 shutdown。
- 合并、时长、字幕偏移或兼容性变化必须跑 `make test-media`，确认 FFmpeg 与 ffprobe 存在，不能把 skip 当通过。
- Stash 阶段改变覆盖 scan/generate/identify/verify、上游失败与已完成 job 恢复；不使用真实媒体库做默认回归。
