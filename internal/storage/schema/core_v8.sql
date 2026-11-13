CREATE TABLE downloader_cleanup_state (
 id INTEGER PRIMARY KEY CHECK(id=1),
 payload_json TEXT NOT NULL CHECK(json_valid(payload_json))
) STRICT;
INSERT INTO downloader_cleanup_state(id,payload_json) VALUES(1,'{}');
DROP INDEX capacity_space_pending;
