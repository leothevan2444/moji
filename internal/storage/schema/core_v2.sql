-- The resolved download location is an ingest input, not merely a live qB
-- metric. It must survive discarding runtime.db and restoring a core backup.
ALTER TABLE task_downloads ADD COLUMN content_path TEXT NOT NULL DEFAULT '';
