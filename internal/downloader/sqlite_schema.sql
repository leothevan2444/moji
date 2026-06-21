PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS task_store_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  query TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'added', 'downloading', 'completed', 'failed')),

  torrent_url TEXT NOT NULL DEFAULT '',
  save_path TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '',

  torrent_hash TEXT NOT NULL DEFAULT '',
  torrent_name TEXT NOT NULL DEFAULT '',
  progress REAL NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 1),
  qbittorrent_state TEXT NOT NULL DEFAULT '',
  content_path TEXT NOT NULL DEFAULT '',

  completed_at TEXT,
  stash_mode TEXT NOT NULL DEFAULT '',
  stash_source_path TEXT NOT NULL DEFAULT '',
  stash_transfer_action TEXT NOT NULL DEFAULT '',
  stash_transfer_path TEXT NOT NULL DEFAULT '',
  stash_transfer_status TEXT NOT NULL DEFAULT '' CHECK (stash_transfer_status IN ('', 'started', 'completed', 'failed')),
  stash_transfer_error TEXT NOT NULL DEFAULT '',
  stash_job_id TEXT NOT NULL DEFAULT '',
  stash_scan_path TEXT NOT NULL DEFAULT '',
  stash_scan_status TEXT NOT NULL DEFAULT '' CHECK (stash_scan_status IN ('', 'started', 'failed')),
  stash_scan_error TEXT NOT NULL DEFAULT '',
  stash_scan_hint TEXT NOT NULL DEFAULT '',
  stash_scan_started_at TEXT,

  error TEXT NOT NULL DEFAULT '',

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
  WHERE status = 'completed' AND stash_job_id = '' AND stash_scan_status = '';

CREATE TABLE IF NOT EXISTS task_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  level TEXT NOT NULL DEFAULT 'info' CHECK (level IN ('debug', 'info', 'warn', 'error')),
  message TEXT NOT NULL,
  old_status TEXT NOT NULL DEFAULT '',
  new_status TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_task_events_task_created_at
  ON task_events (task_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_task_events_created_at
  ON task_events (created_at DESC);

INSERT INTO task_store_meta (key, value)
VALUES ('schema_version', '2')
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

CREATE TABLE IF NOT EXISTS subscription_performer_state (
  performer_id TEXT PRIMARY KEY,
  last_checked_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
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
  release_date TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  query TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  seen_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET DEFAULT
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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  performer_id TEXT NOT NULL,
  release_id INTEGER NOT NULL,
  linked_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (performer_id) REFERENCES subscription_performer_state(performer_id) ON DELETE CASCADE,
  FOREIGN KEY (release_id) REFERENCES subscription_release_entities(id) ON DELETE CASCADE
) STRICT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_performer_releases_unique
  ON subscription_performer_releases (performer_id, release_id);

CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_performer_linked_at
  ON subscription_performer_releases (performer_id, linked_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_performer_releases_release_id
  ON subscription_performer_releases (release_id);
