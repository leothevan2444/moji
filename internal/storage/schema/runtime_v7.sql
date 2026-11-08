-- One authoritative record; indexed columns are derived from its JSON payload.
CREATE TABLE indexer_executions (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 message TEXT NOT NULL CHECK(json_valid(message)),
 instance_id TEXT GENERATED ALWAYS AS (json_extract(message,'$.InstanceID')) STORED NOT NULL,
 operation TEXT GENERATED ALWAYS AS (json_extract(message,'$.Operation')) STORED NOT NULL,
 correlation_id TEXT GENERATED ALWAYS AS (json_extract(message,'$.Report.CorrelationID')) STORED NOT NULL,
 started_at TEXT GENERATED ALWAYS AS (json_extract(message,'$.StartedAt')) STORED NOT NULL,
 finished_at TEXT GENERATED ALWAYS AS (json_extract(message,'$.FinishedAt')) STORED NOT NULL,
 outcome TEXT GENERATED ALWAYS AS (CASE WHEN json_extract(message,'$.Report.Code')='INDEXER_CANCELLED' THEN 'CANCELLED' WHEN coalesce(json_extract(message,'$.Report.Code'),'')<>'' OR json_extract(message,'$.Report.Partial')=1 THEN 'FAILED' ELSE 'SUCCEEDED' END) STORED NOT NULL,
 eligible INTEGER GENERATED ALWAYS AS (json_extract(message,'$.Executed')=1 AND coalesce(json_extract(message,'$.Report.CacheHit'),0)=0 AND coalesce(json_extract(message,'$.Report.Code'),'')<>'INDEXER_CANCELLED') STORED NOT NULL,
 UNIQUE(instance_id,operation,correlation_id)
) STRICT;
CREATE INDEX indexer_executions_period ON indexer_executions(finished_at,operation) WHERE eligible=1;
CREATE INDEX indexer_executions_recent ON indexer_executions(instance_id,started_at DESC,id DESC) WHERE eligible=1;
CREATE INDEX indexer_executions_outcome ON indexer_executions(outcome,id DESC);
