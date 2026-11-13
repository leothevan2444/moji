ALTER TABLE download_policies DROP COLUMN ratio;
ALTER TABLE download_policies DROP COLUMN seed_hours;
ALTER TABLE download_policies DROP COLUMN match_all;
ALTER TABLE download_policies ADD COLUMN capacity_enabled INTEGER NOT NULL DEFAULT 0 CHECK(capacity_enabled IN (0,1));
ALTER TABLE download_policies ADD COLUMN min_free_gb REAL NOT NULL DEFAULT 50 CHECK(min_free_gb>0);
ALTER TABLE download_policies ADD COLUMN capacity_order TEXT NOT NULL DEFAULT 'DISK_USAGE' CHECK(capacity_order IN ('DISK_USAGE','RATIO','SEED_TIME'));
CREATE INDEX capacity_space_pending ON external_commands(json_extract(payload_json,'$.capacity_space_before')) WHERE json_extract(payload_json,'$.capacity_space_before') IS NOT NULL AND coalesce(json_extract(payload_json,'$.capacity_space_confirmed'),0)=0;
