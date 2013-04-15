package logistics

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"time"
)

type UserLogisticsRef struct {
	Id                    int64
	user                  string
	LogisticsInfoEntityId int64
	Subscribe             int //1: subscribe, 2: unsubscribe
}

type LogisticsInfoEntity struct {
	Id             int64
	LogisticsId    string
	Company        string
	State          int //-1: unknown, 0: deliverying, 1: sent, 2: Problem package, 3: Received, 4: Returned
	LastUpdateTime int64
}

type LogisticsRecordEntity struct {
	Id                    int64
	LogisticsInfoEntityId int64
	Context               string
	Time                  int64
}

type LogisticsDb struct {
	dbConn *sql.DB
}

func NewLogisticsDb(dbFile string) (*LogisticsDb, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}
	logisticsDb := &LogisticsDb{}
	logisticsDb.dbConn = db
	return logisticsDb, nil
}

func (self *LogisticsDb) Close() {
	self.dbConn.Close()
}

const (
	GETLOGISTICSINFO_SQL = "select l.id, l.l.logistics_id, l.company, l.state, l.last_upd_time from logistics_info l where l.logistics_id = ? and l.company = ?"

	GETLOGISTICSINFOBYID_SQL = "select l.id, l.l.logistics_id, l.company, l.state, l.last_upd_time from logistics_info l where l.id = ?"

	INSERTRECORD_SQL = "insert into logistics_records(logistics_info_id, context, time) values(?, ?, ?)"

	UPDATELOGISTICSINFO_SQL = "update logistics_info  set state = ? ,last_upd_time = ? where id = ?"
)

func (self *LogisticsDb) GetLogisticsInfoByIdCompany(logisticsId, company string) (*LogisticsInfoEntity, error) {

	getLogisticsStmt, prepareErr1 := self.dbConn.Prepare(GETLOGISTICSINFO_SQL)
	if prepareErr1 != nil {
		return nil, prepareErr1
	}
	defer getLogisticsStmt.Close()
	var id int64 = -1
	var lId string
	var com string
	var state int
	var lastUpdateTime int64
	row := getLogisticsStmt.QueryRow(logisticsId, company)
	rowErr := row.Scan(&id, &lId, &com, &state, &lastUpdateTime)
	if rowErr != nil {
		if rowErr == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, rowErr
		}
	}
	return &LogisticsInfoEntity{id, lId, com, state, lastUpdateTime}, nil
}

//add new LogisticsInfoEntity, will return the id if exist or the new LogisticsInfoEntity's id
func (self *LogisticsDb) AddLogisticsInfo(entity *LogisticsInfoEntity) (*LogisticsInfoEntity, error) {
	l, getError := self.GetLogisticsInfoByIdCompany(entity.LogisticsId, entity.Company)
	if getError != nil {
		return nil, getError
	}
	if l != nil {
		return l, nil
	}
	//not exist
	addLogisticStmt, prepareErr2 := self.dbConn.Prepare("insert into logistics_info(logistics_id, company, state, last_upd_time) values(?, ?, ?, ?)")
	if prepareErr2 != nil {
		return nil, prepareErr2
	}
	defer addLogisticStmt.Close()
	result, addErr := addLogisticStmt.Exec(entity.LogisticsId, entity.Company, -1, -1) // unknown state
	if addErr != nil {
		return nil, addErr
	}
	lastId, lastErr := result.LastInsertId()
	if lastErr != nil {
		return nil, lastErr
	}
	return &LogisticsInfoEntity{lastId, entity.LogisticsId, entity.Company, -1, -1}, nil
}

func (self *LogisticsDb) GetLogisticsInfoByLogisticsEntityId(entityId int64) (*LogisticsInfoEntity, error) {
	getLogisticsStmt, prepareErr1 := self.dbConn.Prepare(GETLOGISTICSINFOBYID_SQL)
	if prepareErr1 != nil {
		return nil, prepareErr1
	}
	defer getLogisticsStmt.Close()
	var id int64 = -1
	var lId string
	var com string
	var state int
	var lastUpdateTime int64
	row := getLogisticsStmt.QueryRow(entityId)
	rowErr := row.Scan(&id, &lId, &com, &state, &lastUpdateTime)
	if rowErr != nil {
		if rowErr == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, rowErr
		}
	}
	return &LogisticsInfoEntity{id, lId, com, state, lastUpdateTime}, nil
}

func (self *LogisticsDb) UpdateLogisticsStatus(logisticsEntityId int64) error {
	logisticsEntity, getError := self.GetLogisticsInfoByLogisticsEntityId(logisticsEntityId)
	if getError != nil {
		return getError
	}
	if logisticsEntity == nil {
		return errors.New(fmt.Sprintf("LogisticsInfo[%d] not exist!", logisticsEntityId))
	}
	logisticsInfo, queryErr := Query(logisticsEntity.Company, logisticsEntity.LogisticsId)
	if queryErr != nil {
		return queryErr
	}
	if logisticsInfo.Status != "200" {
		// TODO don't update invalid logistics id again
		return errors.New("Query logistics error: " + logisticsInfo.Message)
	}

	last := logisticsEntity.LastUpdateTime
	tx, beginErr := self.dbConn.Begin()
	if beginErr != nil {
		return beginErr
	}
	s, convertErr := strconv.Atoi(logisticsInfo.State)
	if convertErr != nil {
		return convertErr
	}
	logisticsEntity.State = s
	logisticsEntity.LastUpdateTime = time.Now().Unix()
	updateErr := self.UpdateLogisticsInfoEntity(tx, logisticsEntity)
	if updateErr != nil {
		return updateErr
	}
	for _, rec := range logisticsInfo.Data {
		recTime, parseErr := time.Parse("2006-01-02 15:04:05", rec.Time)
		if parseErr != nil {
			return parseErr
		}
		rT := recTime.Unix()
		if rT > last {
			// new record TODO may need to return
			recEntity := &LogisticsRecordEntity{LogisticsInfoEntityId: logisticsEntityId,
				Context: rec.Context,
				Time:    rT}
			insertErr := self.InsertLogisticsRec(tx, recEntity)
			if insertErr != nil {
				return insertErr
			}
		}
	}

	tx.Commit()
	return nil
}
func (self *LogisticsDb) UpdateLogisticsInfoEntity(tx *sql.Tx, entity *LogisticsInfoEntity) error {
	stme, prepareErr2 := tx.Prepare(UPDATELOGISTICSINFO_SQL)
	if prepareErr2 != nil {
		return prepareErr2
	}
	defer stme.Close()
	_, err := stme.Exec(entity.State, entity.LastUpdateTime, entity.Id)
	if err != nil {
		return err
	}
	return nil
}

func (self *LogisticsDb) InsertLogisticsRec(tx *sql.Tx, rec *LogisticsRecordEntity) error {
	addRecordStmt, prepareErr2 := tx.Prepare(INSERTRECORD_SQL)
	if prepareErr2 != nil {
		return prepareErr2
	}
	defer addRecordStmt.Close()
	_, addErr := addRecordStmt.Exec(rec.LogisticsInfoEntityId, rec.Context, rec.Time)
	if addErr != nil {
		return addErr
	}
	return nil
}
