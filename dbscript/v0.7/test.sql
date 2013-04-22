insert into logistics_info(logistics_id, company) values("123", "yunda")


select * from logistics_info l 

select l.id from logistics_info l where l.logistics_id = 123 and l.company = 'yunda'


insert into logistics_records(logistics_info_id, context, time) values("321", "message content", 999)

delete from logistics_record_entity

update logistics_info  set state = 1 , last_upd_time = 1 where id = 3