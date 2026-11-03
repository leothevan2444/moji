CREATE TABLE task_indexer_sources (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 payload_version INTEGER NOT NULL CHECK(payload_version=1),
 payload_json TEXT NOT NULL CHECK(json_valid(payload_json))
) STRICT;
