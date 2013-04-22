SELECT strftime('%s','now') from logistics_info_entity

    UPDATE logistics_info_entity
       SET crt_date = strftime('%s','now') 
     WHERE id = 1;
     
 INSERT INTO logistics_info_entity