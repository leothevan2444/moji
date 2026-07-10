PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS task_store_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT 'MANUAL' CHECK (source IN ('MANUAL', 'SEARCH', 'SUBSCRIPTION')),
  code TEXT NOT NULL DEFAULT '',
  stage TEXT NOT NULL CHECK (stage IN ('SOURCING', 'DOWNLOADING', 'PENDING_INGEST', 'TRANSFERRING', 'SCANNING', 'COMPLETED')),
  stage_status TEXT NOT NULL DEFAULT 'PENDING' CHECK (stage_status IN ('PENDING', 'RUNNING', 'BLOCKED', 'DONE')),
  stage_error_code TEXT,
  stage_error_message TEXT,

  torrent_url TEXT NOT NULL DEFAULT '',
  save_path TEXT,
  category TEXT,
  tags TEXT,

  torrent_identity_hash TEXT,
  torrent_identity_magnet TEXT,
  torrent_hash TEXT,
  torrent_name TEXT,
  progress REAL NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 1),
  qbittorrent_state TEXT,
  content_path TEXT,

  download_completed_at TEXT,
  delivery_mode TEXT,
  moji_source_path TEXT,
  transfer_action TEXT,
  moji_transfer_path TEXT,
  transfer_error TEXT,
  stash_scan_job_id TEXT,
  stash_scan_path TEXT,
  stash_scan_error TEXT,
  stash_scan_hint TEXT,
  stash_scan_started_at TEXT,

  selected_title TEXT NOT NULL DEFAULT '',
  selected_tracker TEXT NOT NULL DEFAULT '',
  selected_info_hash TEXT NOT NULL DEFAULT '',
  selected_link TEXT NOT NULL DEFAULT '',
  selected_magnet_uri TEXT NOT NULL DEFAULT '',
  selected_size INTEGER NOT NULL DEFAULT 0,
  selected_seeders INTEGER NOT NULL DEFAULT 0,
  selected_peers INTEGER NOT NULL DEFAULT 0,

  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX IF NOT EXISTS idx_tasks_stage_created_at
  ON tasks (stage, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_updated_at
  ON tasks (updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_download_completed_at
  ON tasks (download_completed_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_torrent_hash
  ON tasks (torrent_hash);

CREATE INDEX IF NOT EXISTS idx_tasks_selected_info_hash
  ON tasks (selected_info_hash);

CREATE INDEX IF NOT EXISTS idx_tasks_stash_scan_job_id
  ON tasks (stash_scan_job_id);

CREATE INDEX IF NOT EXISTS idx_tasks_scan_queue
  ON tasks (updated_at DESC)
  WHERE stage = 'PENDING_INGEST' AND stage_status = 'PENDING' AND stash_scan_job_id IS NULL;

CREATE TABLE IF NOT EXISTS task_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error')),
  message TEXT NOT NULL,
  old_stage TEXT NOT NULL DEFAULT '',
  new_stage TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_task_events_task_created_at
  ON task_events (task_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_task_events_created_at
  ON task_events (created_at DESC);

INSERT INTO task_store_meta (key, value)
VALUES ('schema_version', '7')
ON CONFLICT(key) DO UPDATE SET value = excluded.value;
