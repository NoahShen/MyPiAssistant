drop table user_logistics_ref;

create table user_logistics_ref (
  id                integer PRIMARY KEY,
  user              varchar(255),
  logistics_info_id integer,
  subscribe         integer
);

drop table logistics_info;

create table logistics_info (
  id            integer PRIMARY KEY,
  logistics_id  varchar(255),
  company       varchar(255),
  state         integer, --0：在途中, 1：已发货， 2：疑难件， 3：已签收， 4：已退货。
  last_upd_time integer
);

drop table logistics_records;

create table logistics_records (
  id                integer  PRIMARY KEY,
  logistics_info_id integer,
  context           text,
  time              integer
);


