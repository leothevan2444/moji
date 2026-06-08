# Stash 集成

Moji 会在完成下载后把任务交给 Stash。

1. 优先使用 content path。
2. 其次回退到 save path。
3. 再回退到默认 library path。

## 失败处理

- 扫描失败会记录到任务。
- 后台不会无限重试。
