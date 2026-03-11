create table if not exists annotation_definitions (
    attribute       text not null,
    value           text not null,
    primary key (attribute, value)
);