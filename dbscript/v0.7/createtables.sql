drop table user_logistics_ref;

create table user_logistics_ref (
  id                       integer PRIMARY KEY,
  username                 varchar(255),
  logistics_info_entity_id integer,
  logistics_name           varchar(255),
  subscribe                integer
);

drop table logistics_info_entity;

create table logistics_info_entity (
  id               integer PRIMARY KEY,
  logistics_id     varchar(255),
  company          varchar(255),
  state            integer, --0：在途中, 1：已发货， 2：疑难件， 3：已签收， 4：已退货。
  message          varchar(255),
  last_update_time integer
);

drop table logistics_record_entity;

create table logistics_record_entity (
  id                integer  PRIMARY KEY,
  logistics_info_entity_id integer,
  context           text,
  time              integer
);


