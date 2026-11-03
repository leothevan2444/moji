CREATE TABLE indexer_health (
 instance_id TEXT NOT NULL,
 config_revision TEXT NOT NULL,
 bundle_revision TEXT NOT NULL,
 operation TEXT NOT NULL CHECK(operation IN ('SEARCH','FETCH')),
 started_at TEXT NOT NULL,
 finished_at TEXT NOT NULL,
 error_code TEXT NOT NULL,
 correlation_id TEXT NOT NULL,
 retry_at TEXT,
 PRIMARY KEY(instance_id,config_revision,bundle_revision,operation)
) STRICT;
CREATE INDEX indexer_health_age ON indexer_health(finished_at);
