package logistics

import (
	"database/sql"
	"errors"
	"github.com/NoahShen/beedb"
	_ "github.com/NoahShen/go-sqlite3"
	"strconv"
)

type UserLogisticsRef struct {
	Id                    int `PK`
	Username              string
	LogisticsInfoEntityId int
	LogisticsName         string
	Subscribe             int //1: subscribe, 2: unsubscribe
	CrtDate               int64
	UpdDate               int64
}

type LogisticsInfoEntity struct {
	Id             int `PK`
	LogisticsId    string
	Company        string
	State          int //-1: unknown, 0: deliverying, 1: sent, 2: Problem package, 3: Received, 4: Returned
	Message        string
	LastUpdateTime int64
	CrtDate        int64
	UpdDate        int64
}

type LogisticsRecordEntity struct {
	Id                    int `PK`
	LogisticsInfoEntityId int
	Context               string
	Time                  int64
	CrtDate               int64
	UpdDate               int64
}

type LogisticsDb struct {
	dbConn *sql.DB
	orm    beedb.Model
}

func NewLogisticsDb(dbFile string) (*LogisticsDb, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	logisticsDb := &LogisticsDb{}
	logisticsDb.dbConn = db
	logisticsDb.orm = beedb.New(db)
	return logisticsDb, nil
}

func (self *LogisticsDb) Close() error {
	return self.dbConn.Close()
}

func (self *LogisticsDb) SaveLogisticsInfo(entity *LogisticsInfoEntity) error {
	return self.orm.Save(entity)
}

func (self *LogisticsDb) SaveUserLogisticsRef(entity *UserLogisticsRef) error {
	return self.orm.Save(entity)
}

func (self *LogisticsDb) SaveLogisticsRecord(entity *LogisticsRecordEntity) error {
	return self.orm.Save(entity)
}

func (self *LogisticsDb) GetLogisticsRecords(logisticsInfoEntityId int) ([]LogisticsRecordEntity, error) {
	var entities []LogisticsRecordEntity
	err := self.orm.Where("logistics_info_entity_id = ?", logisticsInfoEntityId).
		FindAll(&entities) // not Find()
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (self *LogisticsDb) GetAllUserLogisticsRefs(username string, limit int) ([]UserLogisticsRef, error) {
	var entities []UserLogisticsRef
	err := self.orm.Where("username = ? and subscribe = ?", username, 1).
		OrderBy("upd_date desc").
		Limit(limit).
		FindAll(&entities) // not Find()
	return entities, err
}

func (self *LogisticsDb) GetUserLogisticsRef(username string, logisticsInfoEntityId int) (*UserLogisticsRef, error) {
	var entities []UserLogisticsRef
	err := self.orm.Where("username = ? and logistics_info_entity_id = ?", username, logisticsInfoEntityId).
		FindAll(&entities) // not Find()
	if err != nil {
		return nil, err
	}
	l := len(entities)
	if l == 1 {
		return &entities[0], nil
	} else if l > 1 {
		return nil, errors.New("More than one record")
	}
	return nil, nil
}

func (self *LogisticsDb) GetSubUserLogisticsRefs(logisticsInfoEntityId int) ([]UserLogisticsRef, error) {
	var entities []UserLogisticsRef
	err := self.orm.Where("logistics_info_entity_id = ? and subscribe = ?", logisticsInfoEntityId, 1).
		FindAll(&entities) // not Find()
	return entities, err
}

func (self *LogisticsDb) GetLogisticsInfoByIdCompany(logisticsId, company string) (*LogisticsInfoEntity, error) {
	var entities []LogisticsInfoEntity
	err := self.orm.Where("logistics_id = ? and company = ?", logisticsId, company).
		FindAll(&entities) // not use Find()
	if err != nil {
		return nil, err
	}
	l := len(entities)
	if l == 1 {
		return &entities[0], nil
	} else if l > 1 {
		return nil, errors.New("More than one record")
	}
	return nil, nil
}

func (self *LogisticsDb) GetLogisticsInfoByEntityId(entityId int) (*LogisticsInfoEntity, error) {
	var entities []LogisticsInfoEntity
	err := self.orm.Where("id=?", entityId).FindAll(&entities) // not Find()
	if err != nil {
		return nil, err
	}
	l := len(entities)
	if l == 1 {
		return &entities[0], nil
	} else if l > 1 {
		return nil, errors.New("More than one record")
	}
	return nil, nil
}

func (self *LogisticsDb) GetAllDeliveringLogistics(username string) ([]UserLogisticsRef, error) {
	refMaps, findErr := self.orm.SetTable("user_logistics_ref r").
		Join("LEFT", "logistics_info_entity l", "r.logistics_info_entity_id = l.id").
		Where("r.subscribe = ? and r.username = ? and l.state in (-1, 0, 1)", 1, username).
		Select("r.id, r.username, r.logistics_info_entity_id, r.logistics_name, r.subscribe, r.crt_date, r.upd_date").
		FindMap()
	if findErr != nil {
		return nil, findErr
	}
	refs := make([]UserLogisticsRef, 0)
	for _, entityMap := range refMaps {
		id, _ := strconv.Atoi(string(entityMap["id"]))
		username := string(entityMap["username"])
		logisticsEntityId, _ := strconv.Atoi(string(entityMap["logistics_info_entity_id"]))
		logisticsName := string(entityMap["logistics_name"])
		subscribe, _ := strconv.Atoi(string(entityMap["subscribe"]))
		crtDate, _ := strconv.ParseInt(string(entityMap["crt_date"]), 10, 0)
		updDate, _ := strconv.ParseInt(string(entityMap["upd_date"]), 10, 0)
		r := UserLogisticsRef{id, username, logisticsEntityId, logisticsName, subscribe, crtDate, updDate}
		refs = append(refs, r)
	}
	return refs, nil
}

func (self *LogisticsDb) GetUserLogisticsRefByIdCompany(username, logisticsId, company string) (*UserLogisticsRef, error) {
	refMaps, findErr := self.orm.SetTable("user_logistics_ref r").
		Join("LEFT", "logistics_info_entity l", "r.logistics_info_entity_id = l.id").
		Where("r.username = ? and l.logistics_id  = ? and l.company = ?",
		username, logisticsId, company).
		Select("r.id, r.username, r.logistics_info_entity_id, r.logistics_name, r.subscribe").
		FindMap()
	if findErr != nil {
		return nil, findErr
	}

	l := len(refMaps)
	if l == 1 {
		entityMap := refMaps[0]
		id, _ := strconv.Atoi(string(entityMap["id"]))
		username := string(entityMap["username"])
		logisticsEntityId, _ := strconv.Atoi(string(entityMap["logistics_info_entity_id"]))
		logisticsName := string(entityMap["logistics_name"])
		subscribe, _ := strconv.Atoi(string(entityMap["subscribe"]))
		crtDate, _ := strconv.ParseInt(string(entityMap["crt_date"]), 10, 0)
		updDate, _ := strconv.ParseInt(string(entityMap["upd_date"]), 10, 0)
		r := &UserLogisticsRef{id, username, logisticsEntityId, logisticsName, subscribe, crtDate, updDate}
		return r, nil

	} else if l > 1 {
		return nil, errors.New("More than one record")
	}
	return nil, nil
}

func (self *LogisticsDb) GetUserLogisticsRefByName(username, logisticsName string) (*UserLogisticsRef, error) {
	var entities []UserLogisticsRef
	err := self.orm.Where("username = ? and logistics_name = ? ", username, logisticsName).
		FindAll(&entities) // not Find()
	if err != nil {
		return nil, err
	}
	l := len(entities)
	if l == 1 {
		return &entities[0], nil
	} else if l > 1 {
		return nil, errors.New("More than one record")
	}
	return nil, nil
}

func (self *LogisticsDb) GetUnfinishedLogistic(timeBefore int64, limit int) ([]LogisticsInfoEntity, error) {
	var entities []LogisticsInfoEntity
	err := self.orm.Where("upd_date < strftime('%s','now') - ? and state in (-1, 0, 1)", timeBefore).
		Limit(limit).
		FindAll(&entities)
	return entities, err
}
