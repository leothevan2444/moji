CREATE TABLE task_indexer_selections (
 task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
 payload_version INTEGER NOT NULL CHECK(payload_version=1),
 resource_id TEXT NOT NULL CHECK(length(resource_id) BETWEEN 1 AND 128),
 record_json TEXT CHECK(record_json IS NULL OR json_valid(record_json) AND length(CAST(record_json AS BLOB))<=262144)
) STRICT;
CREATE TRIGGER task_indexer_selection_unpin AFTER DELETE ON task_indexer_selections
BEGIN
 DELETE FROM indexer_definition_refs WHERE owner_kind='task-resource' AND owner_id=OLD.task_id;
END;
