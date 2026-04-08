create table if not exists annotation_definitions (
    attribute       text not null,
    target          text not null CHECK (target IN ('feed', 'post')),
    value           text not null,
    primary key (attribute, target, value)
);