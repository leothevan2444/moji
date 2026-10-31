CREATE TABLE subscription_upcoming_releases (
 release_key TEXT PRIMARY KEY,
 release_date TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('WAITING','UNCONFIRMED','HANDOFF_PENDING','HANDED_OFF')),
 next_check_at TEXT NOT NULL,
 code TEXT NOT NULL,
 title TEXT NOT NULL,
 revision INTEGER NOT NULL DEFAULT 1,
 record_json TEXT NOT NULL
) STRICT;
CREATE INDEX upcoming_due ON subscription_upcoming_releases(state,next_check_at,release_key);
CREATE INDEX upcoming_calendar ON subscription_upcoming_releases(release_date,release_key);
CREATE TABLE subscription_upcoming_performers (
 release_key TEXT NOT NULL REFERENCES subscription_upcoming_releases(release_key) ON DELETE CASCADE,
 performer_id TEXT NOT NULL,
 name TEXT NOT NULL,
 PRIMARY KEY(release_key,performer_id)
) STRICT;
CREATE INDEX upcoming_performer ON subscription_upcoming_performers(performer_id,release_key);
