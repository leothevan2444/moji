CREATE TABLE indexer_sessions (
 session_key TEXT PRIMARY KEY CHECK(length(session_key)=64),
 instance_id TEXT NOT NULL,
 payload_version INTEGER NOT NULL CHECK(payload_version=1),
 payload_json TEXT NOT NULL CHECK(json_valid(payload_json) AND length(CAST(payload_json AS BLOB))<=262144),
 expires_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
) STRICT;
CREATE INDEX indexer_sessions_expiry ON indexer_sessions(expires_at,session_key);
CREATE INDEX indexer_sessions_instance ON indexer_sessions(instance_id,session_key);
