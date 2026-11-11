# 在 Homepage 中显示 Moji 统计

Moji Widget 显示待处理、进行中、等待资源和最近 24 小时入库的任务数，与 Moji 首页口径一致。

## 配置

在 Homepage 配置目录的 `services.yaml` 中加入以下内容。如果已有「媒体」分组，将 Moji 条目放到该分组下。

```yaml
- 媒体:
    - Moji:
        href: http://192.168.1.10:10000
        widget:
          type: customapi
          url: http://moji:10000/api/v1/widget/homepage
          refreshInterval: 30000
          mappings:
            - field: attention
              label: 待处理
              format: number
            - field: active
              label: 进行中
              format: number
            - field: waiting
              label: 等待资源
              format: number
            - field: completedLast24Hours
              label: 24h 入库
              format: number
```

替换两个地址：

- `href`：你用浏览器打开 Moji 的地址。点击 Homepage 中的 Moji 卡片会打开这里。
- `widget.url`：Homepage 所在机器或容器能访问的完整接口地址。示例中的 `moji` 服务名只适用于两者共享 Docker 网络且服务名为 `moji` 的情况。

Homepage 运行在容器中时，`localhost` 指向 Homepage 自身。两者不在同一 Docker 网络时，改用 Homepage 实际能连接的主机名/IP 和端口。Moji 示例部署默认只将主机端口绑定到回环地址；如需从其他机器访问，应先调整部署网络或端口绑定。

保存后刷新 Homepage，确认 Moji 卡片显示四项数字。示例的刷新间隔为 30 秒，单位是毫秒。只想显示其中几项时，删除不需要的 `mappings` 条目；调整条目顺序即可改变显示顺序。

首版使用 Homepage 自带的 `customapi` 类型，不能填写 `type: moji`。配置格式见 [Homepage Custom API 文档](https://gethomepage.dev/widgets/services/customapi/)。

## 数字含义

| 指标 | 含义 |
| --- | --- |
| 待处理 | 需要你确认选择或处理故障的任务 |
| 进行中 | 正在排队、寻源、下载、整理或入库的任务，不包括等待资源和待处理任务 |
| 等待资源 | 暂未取得可用资源，已安排后续自动检查的任务 |
| 24h 入库 | 最近连续 24 小时内完成入库、且当前仍为已完成状态的任务 |

计数单位为 Moji 任务，不是文件数或 Stash 影片数。24h 入库不等于今日入库或下载完成数；删除任务或重新选种后，数字可能减少。

## 无法显示时

| 现象 | 检查方法 |
| --- | --- |
| 无法连接或域名无法解析 | 从 Homepage 所在机器/容器检查 `widget.url`，确认服务名、端口和网络；浏览器能打开 `href` 不代表 Homepage 能访问接口 |
| 返回 404 或 HTML 页面 | 确认 Moji 版本包含 Widget 功能，且 URL 完整包含 `/api/v1/widget/homepage`；检查反向代理路径是否正确 |
| 返回 401/403 | 检查反向代理的访问认证。可在 `widget.headers` 添加代理要求的请求头，或按 Homepage 文档配置 Basic Auth |
| 返回 500/503/504 | Moji 统计暂时无法读取。检查 Moji 服务和日志，使用错误中的关联 ID 定位；不要把失败当作零任务 |
| 四项均为 0 | 请求成功但当前没有对应任务；可到 Moji 首页核对 |

Moji 不额外要求 Widget API key。如果使用反向代理认证，填写代理凭据；不要填写 Stash 或 Jackett 的 API key。
