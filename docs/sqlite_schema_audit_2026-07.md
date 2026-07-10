# Moji SQLite Schema Audit (2026-07)

本次 SQLite 清理已删除以下低价值或陈旧字段：

- `task_events.payload_json`
- `tasks.status`
- `tasks.completed_at`
- `tasks.stash_mode`
- `tasks.stash_source_path`
- `tasks.stash_transfer_action`
- `tasks.stash_transfer_path`
- `tasks.stash_transfer_status`
- `tasks.stash_transfer_error`
- `tasks.stash_job_id`
- `tasks.stash_scan_status`
- `tasks.error`
- `task_events.old_status`
- `task_events.new_status`
- `subscription_release_entities.last_error`
- `subscription_performer_releases.id`
- `subscription_performer_releases.created_at`
- `subscription_performer_releases.updated_at`

以下字段在本轮审计后保留，没有删除：

- `tasks.selected_title`
  原因：任务详情页、任务摘要和下载链路回填逻辑仍依赖候选标题作为展示与匹配兜底值。
- `tasks.selected_tracker`
  原因：候选来源仍是任务快照的一部分，后续排障和候选解释需要它。
- `tasks.selected_link`
  原因：当前仍属于已选候选快照的一部分，删除会让任务回放信息不完整。
- `tasks.selected_info_hash`
  原因：仍用于候选身份快照和部分去重/匹配回退判断。
- `tasks.selected_magnet_uri`
  原因：手动磁链任务和候选快照仍需要保留原始磁链信息。
- `tasks.selected_size`
  原因：候选体积仍用于任务详情展示和选种结果回放。
- `tasks.selected_seeders`
  原因：候选做出选择时的做种数仍是任务解释信息的一部分。
- `tasks.selected_peers`
  原因：候选做出选择时的 peers 信息仍有排障价值。
- `tasks.delivery_mode`
  原因：任务详情页需要直接展示当前任务采用的是 `PATH_MAP` 还是 `TRANSFER`。
- `tasks.moji_source_path`
  原因：路径命名空间已拆分，任务必须显式记录 Moji 实际访问的源路径。
- `tasks.transfer_action`
  原因：`TRANSFER` 模式下复制、移动、符号链接是业务语义的一部分。
- `tasks.moji_transfer_path`
  原因：任务需要保留实际搬运目标，用于排障与详情展示。
- `tasks.stash_scan_job_id`
  原因：扫描 job 轮询与状态闭环都依赖该字段。
- `tasks.stash_scan_path`
  原因：Stash 扫描路径是路径映射闭环中的最终产物，必须持久化。
- `tasks.stash_scan_hint`
  原因：用于向前端解释当前路径映射关系或配置缺失原因。
- `subscription_release_entities.source`
  原因：订阅页仍展示和保留 release 来源语义，且它是 release key 之外的重要解释字段。
