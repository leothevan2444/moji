# Moji SQLite Schema Audit (2026-07)

本次 SQLite 清理已删除以下低价值字段：

- `task_events.payload_json`
- `subscription_release_entities.last_error`
- `subscription_performer_releases.id`
- `subscription_performer_releases.created_at`
- `subscription_performer_releases.updated_at`

以下字段在本轮审计后保留，没有删除：

- `tasks.candidate_title`
  原因：任务详情页、任务摘要和下载链路回填逻辑仍依赖候选标题作为展示与匹配兜底值。
- `tasks.candidate_tracker`
  原因：候选来源仍是任务快照的一部分，后续排障和候选解释需要它。
- `tasks.candidate_link`
  原因：当前仍属于已选候选快照的一部分，删除会让任务回放信息不完整。
- `tasks.candidate_info_hash`
  原因：仍用于候选身份快照和部分去重/匹配回退判断。
- `tasks.candidate_magnet_uri`
  原因：手动磁链任务和候选快照仍需要保留原始磁链信息。
- `subscription_release_entities.source`
  原因：订阅页仍展示和保留 release 来源语义，且它是 release key 之外的重要解释字段。
