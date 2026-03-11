create table if not exists feed_sources (
    source  text not null,
    trusted boolean not null,
    primary key (source)
);