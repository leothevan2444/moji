CREATE TABLE IF NOT EXISTS stashbox_cache_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stashbox_cache_performers (
    endpoint TEXT NOT NULL,
    performer_id TEXT NOT NULL,
    payload_json BLOB NOT NULL,
    fetched_at TEXT NOT NULL,
    last_accessed_at TEXT NOT NULL,
    PRIMARY KEY (endpoint, performer_id)
);

CREATE TABLE IF NOT EXISTS stashbox_cache_scenes (
    endpoint TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    payload_json BLOB NOT NULL,
    PRIMARY KEY (endpoint, scene_id)
);

CREATE TABLE IF NOT EXISTS stashbox_cache_snapshots (
    endpoint TEXT NOT NULL,
    performer_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    remote_count INTEGER NOT NULL,
    complete INTEGER NOT NULL,
    active INTEGER NOT NULL,
    updated_at TEXT NOT NULL,
    last_accessed_at TEXT NOT NULL,
    PRIMARY KEY (endpoint, performer_id, generation)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_stashbox_cache_active_snapshot
ON stashbox_cache_snapshots(endpoint, performer_id) WHERE active = 1;

CREATE TABLE IF NOT EXISTS stashbox_cache_snapshot_pages (
    endpoint TEXT NOT NULL,
    performer_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    page_number INTEGER NOT NULL,
    fetched_at TEXT NOT NULL,
    PRIMARY KEY (endpoint, performer_id, generation, page_number),
    FOREIGN KEY (endpoint, performer_id, generation)
      REFERENCES stashbox_cache_snapshots(endpoint, performer_id, generation) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stashbox_cache_snapshot_items (
    endpoint TEXT NOT NULL,
    performer_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    position INTEGER NOT NULL,
    scene_id TEXT NOT NULL,
    PRIMARY KEY (endpoint, performer_id, generation, position),
    FOREIGN KEY (endpoint, performer_id, generation)
      REFERENCES stashbox_cache_snapshots(endpoint, performer_id, generation) ON DELETE CASCADE
);
