CREATE TABLE torrent_search_runs (
 id TEXT PRIMARY KEY,
 payload_version INTEGER NOT NULL CHECK(payload_version=1),
 state TEXT NOT NULL CHECK(state IN ('QUEUED','RUNNING','COMPLETED','PARTIAL','FAILED','CANCELLED','INTERRUPTED')),
 query_json TEXT NOT NULL CHECK(json_valid(query_json) AND length(CAST(query_json AS BLOB))<=262144),
 task_id TEXT NOT NULL DEFAULT '',
 revision INTEGER NOT NULL DEFAULT 1,
 cancel_requested INTEGER NOT NULL DEFAULT 0 CHECK(cancel_requested IN (0,1)),
 partial INTEGER NOT NULL DEFAULT 0 CHECK(partial IN (0,1)),
 truncated INTEGER NOT NULL DEFAULT 0 CHECK(truncated IN (0,1)),
 error_code TEXT NOT NULL DEFAULT '',
 correlation_id TEXT NOT NULL,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 finished_at TEXT,
 expires_at TEXT NOT NULL
) STRICT;
CREATE INDEX torrent_search_queue ON torrent_search_runs(state,created_at,id);
CREATE INDEX torrent_search_expiry ON torrent_search_runs(expires_at,id);
CREATE TABLE torrent_search_sites (
 search_id TEXT NOT NULL REFERENCES torrent_search_runs(id) ON DELETE CASCADE,
 instance_id TEXT NOT NULL,
 position INTEGER NOT NULL,
 name TEXT NOT NULL,
 kind TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('QUEUED','RUNNING','COMPLETED','FAILED','CANCELLED','INTERRUPTED','SKIPPED')),
 report_json TEXT CHECK(report_json IS NULL OR json_valid(report_json)),
 fetched_at TEXT,
 PRIMARY KEY(search_id,instance_id)
) STRICT;
CREATE TABLE torrent_search_candidates (
 id TEXT PRIMARY KEY,
 search_id TEXT NOT NULL REFERENCES torrent_search_runs(id) ON DELETE CASCADE,
 position INTEGER NOT NULL,
 instance_id TEXT NOT NULL,
 resource_id TEXT NOT NULL,
 payload_version INTEGER NOT NULL CHECK(payload_version=1),
 payload_json TEXT NOT NULL CHECK(json_valid(payload_json) AND length(CAST(payload_json AS BLOB))<=65536),
 size INTEGER,
 seeders INTEGER,
 published_at TEXT,
 UNIQUE(search_id,position)
) STRICT;
CREATE INDEX torrent_search_candidates_page ON torrent_search_candidates(search_id,position);
