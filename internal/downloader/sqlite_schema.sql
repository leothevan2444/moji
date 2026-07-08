PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS task_store_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT 'MANUAL' CHECK (source IN ('MANUAL', 'SEARCH', 'SUBSCRIPTION')),
  query TEXT NOT NULL,
  code TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('pending', 'added', 'downloading', 'completed', 'failed')),

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

  completed_at TEXT,
  stash_mode TEXT,
  stash_source_path TEXT,
  stash_transfer_action TEXT,
  stash_transfer_path TEXT,
  stash_transfer_status TEXT CHECK (stash_transfer_status IN ('started', 'completed', 'failed')),
  stash_transfer_error TEXT,
  stash_job_id TEXT,
  stash_scan_path TEXT,
  stash_scan_status TEXT CHECK (stash_scan_status IN ('started', 'failed')),
  stash_scan_error TEXT,
  stash_scan_hint TEXT,
  stash_scan_started_at TEXT,

  error TEXT,

  candidate_title TEXT NOT NULL DEFAULT '',
  candidate_tracker TEXT NOT NULL DEFAULT '',
  candidate_info_hash TEXT NOT NULL DEFAULT '',
  candidate_link TEXT NOT NULL DEFAULT '',
  candidate_magnet_uri TEXT NOT NULL DEFAULT '',
  candidate_size INTEGER NOT NULL DEFAULT 0,
  candidate_seeders INTEGER NOT NULL DEFAULT 0,
  candidate_peers INTEGER NOT NULL DEFAULT 0,

  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX IF NOT EXISTS idx_tasks_status_created_at
  ON tasks (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_updated_at
  ON tasks (updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_completed_at
  ON tasks (completed_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_torrent_hash
  ON tasks (torrent_hash);

CREATE INDEX IF NOT EXISTS idx_tasks_candidate_info_hash
  ON tasks (candidate_info_hash);

CREATE INDEX IF NOT EXISTS idx_tasks_stash_job_id
  ON tasks (stash_job_id);

CREATE INDEX IF NOT EXISTS idx_tasks_scan_queue
  ON tasks (updated_at DESC)
  WHERE status = 'completed' AND stash_job_id IS NULL AND stash_scan_status IS NULL;

CREATE TABLE IF NOT EXISTS task_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error')),
  message TEXT NOT NULL,
  old_status TEXT NOT NULL DEFAULT '',
  new_status TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_task_events_task_created_at
  ON task_events (task_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_task_events_created_at
  ON task_events (created_at DESC);

INSERT INTO task_store_meta (key, value)
VALUES ('schema_version', '4')
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

CREATE TABLE IF NOT EXISTS subscription_performer_state (
  performer_id TEXT PRIMARY KEY,
  last_checked_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX IF NOT EXISTS idx_subscription_state_last_checked_at
  ON subscription_performer_state (last_checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_state_updated_at
  ON subscription_performer_state (updated_at DESC);

CREATE TABLE IF NOT EXISTS subscription_release_entities (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  release_key TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered', 'pending', 'processed', 'failed')),
  source TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  code TEXT NOT NULL DEFAULT '',
  release_date TEXT,
  url TEXT,
  query TEXT NOT NULL DEFAULT '',
  task_id TEXT,
  performer_count INTEGER NOT NULL DEFAULT 0,
  performer_names TEXT NOT NULL DEFAULT '[]',
  classification TEXT NOT NULL DEFAULT '',
  decision TEXT NOT NULL DEFAULT '',
  decision_reason TEXT NOT NULL DEFAULT '',
  seen_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
) STRICT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_release_entities_key
  ON subscription_release_entities (release_key);

CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_date
  ON subscription_release_entities (release_date DESC, seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_code
  ON subscription_release_entities (code);

CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_status_seen_at
  ON subscription_release_entities (status, seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_release_entities_task_id
  ON subscription_release_entities (task_id);

CREATE TABLE IF NOT EXISTS subscription_performer_releases (
  performer_id TEXT NOT NULL,
  release_id INTEGER NOT NULL,
  linked_at TEXT NOT NULL,
  PRIMARY KEY (performer_id, release_id),
  FOREIGN KEY (performer_id) REFERENCES subscription_performer_state(performer_id) ON DELETE CASCADE,
  FOREIGN KEY (release_id) REFERENCES subscription_release_entities(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_performer_linked_at
  ON subscription_performer_releases (performer_id, linked_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_release_id
  ON subscription_performer_releases (release_id);
