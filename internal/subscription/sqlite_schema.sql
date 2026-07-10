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
