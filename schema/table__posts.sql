DROP TABLE IF EXISTS posts CASCADE;

CREATE TABLE IF NOT EXISTS posts (
    base_url        TEXT NOT NULL,
    feed_url        TEXT NOT NULL,
    post_url        TEXT NOT NULL PRIMARY KEY,
    post_title      TEXT NOT NULL,
    utc_discovered	TIMESTAMPTZ NOT NULL,
    utc_published   TIMESTAMPTZ NOT NULL,
    INDEX posts_utc_discovered_idx (utc_discovered DESC)
) WITH (
    ttl = 'on', 
    ttl_expiration_expression = 'utc_discovered + INTERVAL ''48 hours'''
);