create table if not exists feeds (
    base_url        text not null,
    feed_url        text not null,
    source          text not null,
    utc_submitted   timestamptz not null,
    primary key (base_url, feed_url, source)
);