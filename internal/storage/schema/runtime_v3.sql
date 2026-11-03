CREATE TABLE indexer_resources (
 id TEXT PRIMARY KEY,
 instance_id TEXT NOT NULL,
 bundle_revision TEXT NOT NULL DEFAULT '',
 payload_version INTEGER NOT NULL CHECK(payload_version=1),
 payload_json TEXT NOT NULL CHECK(json_valid(payload_json) AND length(CAST(payload_json AS BLOB))<=262144),
 expires_at TEXT NOT NULL,
 created_at TEXT NOT NULL
) STRICT;
CREATE INDEX indexer_resources_expiry ON indexer_resources(expires_at,id);
CREATE INDEX indexer_resources_bundle ON indexer_resources(bundle_revision,expires_at);
