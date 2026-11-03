CREATE TABLE indexer_network_cooldowns (
 scope_key TEXT PRIMARY KEY CHECK(length(scope_key)=64),
 failures INTEGER NOT NULL CHECK(failures BETWEEN 1 AND 7),
 blocked_until_ms INTEGER NOT NULL,
 expires_at_ms INTEGER NOT NULL
) STRICT;
CREATE INDEX indexer_network_cooldowns_expiry ON indexer_network_cooldowns(expires_at_ms);
