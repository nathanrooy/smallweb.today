create table if not exists feed_states (
    feed_url        text primary key,
    last_checked    timestamptz not null,
    last_modified   text not null default '',
    etag            text not null default ''
) with (
    ttl = 'on', 
    ttl_expiration_expression = 'last_checked + INTERVAL ''48 hours'''
);