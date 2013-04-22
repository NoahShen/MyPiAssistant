-----logistics_info_entity
DROP TRIGGER logistics_info_entity_i_trigger;
CREATE TRIGGER logistics_info_entity_i_trigger
   AFTER INSERT
   ON logistics_info_entity
BEGIN
    UPDATE logistics_info_entity
       SET crt_date = strftime('%s','now') 
     WHERE id = NEW.id;
END;

DROP TRIGGER logistics_info_entity_u_trigger;
CREATE TRIGGER logistics_info_entity_u_trigger
   AFTER UPDATE
   ON logistics_info_entity
BEGIN
    UPDATE logistics_info_entity
       SET upd_date = strftime('%s','now') 
     WHERE id = OLD.id;
END;

-----user_logistics_ref
DROP TRIGGER user_logistics_ref_i_trigger;
CREATE TRIGGER user_logistics_ref_i_trigger
   AFTER INSERT
   ON user_logistics_ref
BEGIN
    UPDATE user_logistics_ref
       SET crt_date = strftime('%s','now') 
     WHERE id = NEW.id;
END;

DROP TRIGGER user_logistics_ref_u_trigger;
CREATE TRIGGER user_logistics_ref_u_trigger
   AFTER UPDATE
   ON user_logistics_ref
BEGIN
    UPDATE user_logistics_ref
       SET upd_date = strftime('%s','now') 
     WHERE id = OLD.id;
END;

-----logistics_record_entity
DROP TRIGGER logistics_record_entity_i_trigger;
CREATE TRIGGER logistics_record_entity_i_trigger
   AFTER INSERT
   ON logistics_record_entity
BEGIN
    UPDATE logistics_record_entity
       SET crt_date = strftime('%s','now') 
     WHERE id = NEW.id;
END;

DROP TRIGGER logistics_record_entity_u_trigger;
CREATE TRIGGER logistics_record_entity_u_trigger
   AFTER UPDATE
   ON logistics_record_entity
BEGIN
    UPDATE logistics_record_entity
       SET upd_date = strftime('%s','now') 
     WHERE id = OLD.id;
END;
