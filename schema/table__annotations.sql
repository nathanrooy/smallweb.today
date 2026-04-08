create table if not exists annotations (
    base_url            text not null,
    target              text not null,
    target_url          text not null,
    annotator           text not null,
    annotation_type     text not null,
    annotation_value    text not null,
    annotation_utc      timestamptz default current_timestamp,
    primary key (base_url, target, target_url, annotator, annotation_type, annotation_value)
);