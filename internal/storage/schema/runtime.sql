CREATE TABLE task_observations (
 task_id TEXT PRIMARY KEY,
 torrent_name TEXT NOT NULL DEFAULT '',
 progress REAL NOT NULL DEFAULT 0 CHECK(progress>=0 AND progress<=1),
 qbittorrent_state TEXT NOT NULL DEFAULT '',content_path TEXT NOT NULL DEFAULT '',
 last_progress_at TEXT,observed_at TEXT NOT NULL
) STRICT;
CREATE TABLE downloader_state (
 key TEXT PRIMARY KEY CHECK(key='snapshot'),
 value TEXT NOT NULL CHECK(json_valid(value)),
 observed_at TEXT NOT NULL
) STRICT;
CREATE TABLE diagnostics (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 task_id TEXT,component TEXT NOT NULL,level TEXT NOT NULL,message TEXT NOT NULL,
 created_at TEXT NOT NULL
) STRICT;
CREATE INDEX diagnostics_time ON diagnostics(created_at,id);
