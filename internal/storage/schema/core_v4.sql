CREATE TABLE indexer_definition_bundles (
 revision TEXT PRIMARY KEY CHECK(length(revision)=40 AND revision NOT GLOB '*[^0-9a-f]*'),
 manifest_json TEXT NOT NULL CHECK(json_valid(manifest_json)),
 manifest_sha256 TEXT NOT NULL CHECK(length(manifest_sha256)=64),
 installed_at TEXT NOT NULL
) STRICT;
CREATE TABLE indexer_definition_files (
 revision TEXT NOT NULL REFERENCES indexer_definition_bundles(revision) ON DELETE CASCADE,
 name TEXT NOT NULL CHECK(length(name)>0 AND name NOT IN ('.','..') AND instr(name,'/')=0 AND instr(name,char(92))=0),
 sha256 TEXT NOT NULL CHECK(length(sha256)=64),
 bytes INTEGER NOT NULL CHECK(bytes>=0),
 PRIMARY KEY(revision,name)
) STRICT;
CREATE TABLE indexer_definition_state (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 active_revision TEXT REFERENCES indexer_definition_bundles(revision),
 previous_revision TEXT REFERENCES indexer_definition_bundles(revision),
 revision INTEGER NOT NULL DEFAULT 1,
 available_revision TEXT NOT NULL DEFAULT '',
 etag TEXT NOT NULL DEFAULT '',
 last_checked_at TEXT,
 check_error TEXT NOT NULL DEFAULT ''
) STRICT;
INSERT INTO indexer_definition_state(singleton) VALUES(1);
CREATE TABLE indexer_definition_jobs (
 id TEXT PRIMARY KEY,
 kind TEXT NOT NULL CHECK(kind IN ('INSTALL','UPDATE','ROLLBACK','REPAIR')),
 state TEXT NOT NULL CHECK(state IN ('QUEUED','DOWNLOADING','VALIDATING','CHECKING_INSTANCES','ACTIVATING','COMPLETED','FAILED','CANCELLED','INTERRUPTED')),
 target_revision TEXT NOT NULL CHECK(length(target_revision)=40 AND target_revision NOT GLOB '*[^0-9a-f]*'),
 expected_active TEXT NOT NULL,
 downloaded_bytes INTEGER NOT NULL DEFAULT 0 CHECK(downloaded_bytes>=0),
 error_code TEXT NOT NULL DEFAULT '',
 correlation_id TEXT NOT NULL,
 started_at TEXT NOT NULL,
 finished_at TEXT,
 revision INTEGER NOT NULL DEFAULT 1,
 CHECK((state IN ('COMPLETED','FAILED','CANCELLED','INTERRUPTED'))=(finished_at IS NOT NULL))
) STRICT;
CREATE UNIQUE INDEX indexer_definition_one_job ON indexer_definition_jobs((1)) WHERE state IN ('QUEUED','DOWNLOADING','VALIDATING','CHECKING_INSTANCES','ACTIVATING');
CREATE INDEX indexer_definition_job_history ON indexer_definition_jobs(started_at DESC,id);
CREATE TABLE indexer_definition_refs (
 owner_kind TEXT NOT NULL,
 owner_id TEXT NOT NULL,
 revision TEXT NOT NULL REFERENCES indexer_definition_bundles(revision),
 PRIMARY KEY(owner_kind,owner_id,revision)
) STRICT;
CREATE TABLE indexer_definition_audit (
 id INTEGER PRIMARY KEY,
 job_id TEXT NOT NULL REFERENCES indexer_definition_jobs(id),
 previous_revision TEXT NOT NULL,
 active_revision TEXT NOT NULL,
 created_at TEXT NOT NULL
) STRICT;
