SELECT strftime('%s','now') from logistics_info_entity

    UPDATE logistics_info_entity
       SET crt_date = strftime('%s','now') 
     WHERE id = 1;
     
 INSERT INTO logistics_info_entity
 
 SELECT  l.* 
     FROM logistics_info_entity l
   WHERE l.last_update_time < strftime('%s','now')  -  600