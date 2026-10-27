CREATE TABLE tasks (
 id TEXT PRIMARY KEY,
 source TEXT NOT NULL CHECK(source IN ('MANUAL','SEARCH','SUBSCRIPTION')),
 code TEXT NOT NULL,
 stage TEXT NOT NULL CHECK(stage IN ('SOURCING','DOWNLOADING','PENDING_INGEST','TRANSFERRING','SCANNING','COMPLETED')),
 stage_status TEXT NOT NULL CHECK(stage_status IN ('PENDING','RUNNING','BLOCKED','DONE')),
 stage_error_code TEXT,
 stage_error_message TEXT,
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
 generation INTEGER NOT NULL DEFAULT 0 CHECK(generation>=0),
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
) STRICT;
CREATE UNIQUE INDEX tasks_code ON tasks(code) WHERE code<>'';
CREATE INDEX tasks_created ON tasks(created_at DESC,id);
CREATE INDEX tasks_stage_created ON tasks(stage,stage_status,created_at,id);
CREATE INDEX tasks_updated ON tasks(updated_at DESC,id);

CREATE TABLE task_downloads (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 torrent_url TEXT NOT NULL DEFAULT '',
 save_path TEXT NOT NULL DEFAULT '',
 category TEXT NOT NULL DEFAULT '',
 tags TEXT NOT NULL DEFAULT '',
 torrent_identity_hash TEXT,
 torrent_identity_magnet TEXT,
 torrent_hash TEXT NOT NULL DEFAULT ''
) STRICT;
CREATE UNIQUE INDEX task_identity_hash ON task_downloads(torrent_identity_hash) WHERE torrent_identity_hash IS NOT NULL;
CREATE UNIQUE INDEX task_identity_magnet ON task_downloads(torrent_identity_magnet) WHERE torrent_identity_magnet IS NOT NULL;
CREATE INDEX task_download_hash ON task_downloads(torrent_hash);
CREATE TABLE task_candidates (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 title TEXT NOT NULL DEFAULT '',tracker TEXT NOT NULL DEFAULT '',info_hash TEXT NOT NULL DEFAULT '',
 link TEXT NOT NULL DEFAULT '',magnet_uri TEXT NOT NULL DEFAULT '',
 size INTEGER NOT NULL DEFAULT 0,seeders INTEGER NOT NULL DEFAULT 0,peers INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE TABLE task_checkpoints (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 payload_version INTEGER NOT NULL DEFAULT 1,
 sourcing_request_json TEXT CHECK(sourcing_request_json IS NULL OR json_valid(sourcing_request_json)),
 restart_state_json TEXT CHECK(restart_state_json IS NULL OR json_valid(restart_state_json)),
 sourcing_last_checked_at TEXT,sourcing_next_check_at TEXT,
 sourcing_completed_at TEXT,download_completed_at TEXT,ingest_completed_at TEXT
) STRICT;
CREATE INDEX task_due ON task_checkpoints(sourcing_next_check_at,task_id) WHERE sourcing_next_check_at IS NOT NULL;
CREATE INDEX task_ingested ON task_checkpoints(ingest_completed_at DESC,task_id) WHERE ingest_completed_at IS NOT NULL;
CREATE TABLE task_file_plans (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 generation INTEGER NOT NULL DEFAULT 0,
 payload_version INTEGER NOT NULL DEFAULT 1,
 confirmed INTEGER NOT NULL CHECK(confirmed IN (0,1)),
 keep_subtitles INTEGER NOT NULL CHECK(keep_subtitles IN (0,1)),
 pause_requested INTEGER NOT NULL CHECK(pause_requested IN (0,1)),
 applied INTEGER NOT NULL CHECK(applied IN (0,1))
) STRICT;
CREATE TABLE task_files (
 task_id TEXT NOT NULL REFERENCES task_file_plans(task_id) ON DELETE CASCADE,
 file_index INTEGER NOT NULL CHECK(file_index>=0),
 position INTEGER NOT NULL CHECK(position>=0),
 path TEXT NOT NULL,size INTEGER NOT NULL CHECK(size>=0),
 kind TEXT NOT NULL CHECK(kind IN ('MAIN','SUBTITLE','DISCARD','UNKNOWN_VIDEO')),
 part INTEGER NOT NULL DEFAULT 0,selected INTEGER NOT NULL CHECK(selected IN (0,1)),reason TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(task_id,file_index),UNIQUE(task_id,position)
) STRICT;
CREATE TABLE task_attachments (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 torrent_file_data BLOB NOT NULL CHECK(length(torrent_file_data)>0 AND length(torrent_file_data)<=8388608)
) STRICT;
CREATE TABLE task_ingest (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 moji_source_path TEXT NOT NULL DEFAULT '',transfer_action TEXT NOT NULL DEFAULT '',moji_transfer_path TEXT NOT NULL DEFAULT '',transfer_error TEXT NOT NULL DEFAULT '',
 stash_scan_job_id TEXT NOT NULL DEFAULT '',stash_ingest_phase TEXT NOT NULL DEFAULT '',stash_generate_job_id TEXT NOT NULL DEFAULT '',stash_identify_job_id TEXT NOT NULL DEFAULT '',
 stash_scene_ids_json TEXT CHECK(stash_scene_ids_json IS NULL OR json_valid(stash_scene_ids_json)),
 stash_scene_recovery_count INTEGER NOT NULL DEFAULT 0,
 stash_scan_path TEXT NOT NULL DEFAULT '',stash_scan_error TEXT NOT NULL DEFAULT '',stash_scan_hint TEXT NOT NULL DEFAULT '',stash_scan_started_at TEXT
) STRICT;
CREATE INDEX task_scan_job ON task_ingest(stash_scan_job_id) WHERE stash_scan_job_id<>'';

CREATE TABLE audit_events (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 operation_id TEXT UNIQUE,
 task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
 subject_id TEXT NOT NULL,
 event_type TEXT NOT NULL,
 level TEXT NOT NULL CHECK(level IN ('debug','info','warn','error')),
 message TEXT NOT NULL,old_stage TEXT NOT NULL DEFAULT '',new_stage TEXT NOT NULL DEFAULT '',
 payload_version INTEGER NOT NULL DEFAULT 1,payload_json TEXT CHECK(payload_json IS NULL OR json_valid(payload_json)),
 created_at TEXT NOT NULL
) STRICT;
CREATE INDEX audit_subject_time ON audit_events(subject_id,created_at DESC,id DESC);
CREATE INDEX audit_time ON audit_events(created_at,id);
CREATE TABLE history_archives (
 id TEXT PRIMARY KEY,path TEXT NOT NULL,sha256 TEXT NOT NULL,source_rows INTEGER NOT NULL,
 retained_rows INTEGER NOT NULL,created_at TEXT NOT NULL
) STRICT;
CREATE TABLE download_policies (
 id INTEGER PRIMARY KEY CHECK(id=1),
 after_ingest INTEGER NOT NULL DEFAULT 0 CHECK(after_ingest IN (0,1)),
 enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0,1)),
 ratio REAL NOT NULL DEFAULT 0 CHECK(ratio>=0),seed_hours REAL NOT NULL DEFAULT 0 CHECK(seed_hours>=0),
 match_all INTEGER NOT NULL DEFAULT 0 CHECK(match_all IN (0,1)),delete_files INTEGER NOT NULL DEFAULT 1 CHECK(delete_files IN (0,1)),
 revision INTEGER NOT NULL DEFAULT 1
) STRICT;
INSERT INTO download_policies(id) VALUES(1);
CREATE TABLE external_commands (
 id TEXT PRIMARY KEY,
 task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
 target_hash TEXT NOT NULL,
 kind TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('PENDING','SENT','CONFIRMED','FAILED','UNCERTAIN')),
 payload_version INTEGER NOT NULL DEFAULT 1,
 payload_json TEXT NOT NULL CHECK(json_valid(payload_json)),
 last_error TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL,updated_at TEXT NOT NULL
) STRICT;
CREATE INDEX external_commands_pending ON external_commands(state,updated_at,id) WHERE state IN ('PENDING','SENT','UNCERTAIN');
CREATE INDEX external_commands_hash ON external_commands(target_hash,created_at DESC);
CREATE TABLE work_items (
 id TEXT PRIMARY KEY,
 task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
 kind TEXT NOT NULL,generation INTEGER NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('PENDING','RUNNING','DONE','FAILED')),
 due_at TEXT NOT NULL,lease_until TEXT,attempt INTEGER NOT NULL DEFAULT 0,
 last_error TEXT NOT NULL DEFAULT '',updated_at TEXT NOT NULL,
 UNIQUE(task_id,kind,generation)
) STRICT;
CREATE INDEX work_items_due ON work_items(status,due_at,id) WHERE status IN ('PENDING','RUNNING');

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

CREATE TABLE subscription_release_reviews (
 release_key TEXT PRIMARY KEY,
 status TEXT NOT NULL,
 code TEXT NOT NULL DEFAULT '',title TEXT NOT NULL DEFAULT '',source TEXT NOT NULL DEFAULT '',
 release_date TEXT,task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
 decision TEXT NOT NULL DEFAULT '',reviewed_at TEXT,seen_at TEXT NOT NULL,
 payload_version INTEGER NOT NULL DEFAULT 1,
 record_json TEXT NOT NULL CHECK(json_valid(record_json))
) STRICT;
CREATE INDEX release_review_status_date ON subscription_release_reviews(status,release_date DESC,release_key);
CREATE INDEX release_review_code ON subscription_release_reviews(code);
CREATE TABLE subscription_review_performers (
 release_key TEXT NOT NULL REFERENCES subscription_release_reviews(release_key) ON DELETE CASCADE,
 performer_id TEXT NOT NULL,name TEXT NOT NULL DEFAULT '',position INTEGER NOT NULL,
 PRIMARY KEY(release_key,performer_id)
) STRICT;
CREATE INDEX review_performer_release ON subscription_review_performers(performer_id,release_key);
CREATE TABLE subscription_check_runs (
 id TEXT PRIMARY KEY,
 trigger TEXT NOT NULL CHECK(trigger IN ('MANUAL','SCHEDULED')),
 scope TEXT NOT NULL,
 status TEXT NOT NULL,
 started_at TEXT NOT NULL,finished_at TEXT,error TEXT NOT NULL DEFAULT '',
 payload_version INTEGER NOT NULL DEFAULT 1
) STRICT;
CREATE INDEX check_runs_started ON subscription_check_runs(started_at DESC,id);
CREATE TABLE subscription_check_items (
 run_id TEXT NOT NULL REFERENCES subscription_check_runs(id) ON DELETE CASCADE,
 performer_id TEXT NOT NULL,name TEXT NOT NULL DEFAULT '',position INTEGER NOT NULL,
 status TEXT NOT NULL,started_at TEXT,finished_at TEXT,error TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(run_id,performer_id),UNIQUE(run_id,position)
) STRICT;
