create table if not exists post_annotations (
    base_url            text not null,
    post_url            text not null,
    annotator           text not null,
    annotation_type     text not null,
    annotation_value    text not null,
    annotation_utc      timestamptz default current_timestamp,
    primary key (post_url, annotation_type, annotation_value)
);