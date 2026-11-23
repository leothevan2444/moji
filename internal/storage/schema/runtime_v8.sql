CREATE TABLE torrent_selection_previews (
 id TEXT PRIMARY KEY,
 search_id TEXT NOT NULL REFERENCES torrent_search_runs(id) ON DELETE CASCADE,
 state TEXT NOT NULL CHECK(state IN ('QUEUED','RUNNING','COMPLETED','FAILED','CANCELLED','INTERRUPTED')),
 request_json TEXT NOT NULL CHECK(json_valid(request_json)),
 result_json TEXT CHECK(result_json IS NULL OR json_valid(result_json)),
 error_code TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 expires_at TEXT NOT NULL
) STRICT;
CREATE INDEX torrent_selection_preview_search ON torrent_selection_previews(search_id,created_at);
