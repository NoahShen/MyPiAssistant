package aqi

import (
	"database/sql"
	_ "github.com/NoahShen/go-sqlite3"
	"github.com/NoahShen/gorp"
	"log"
	"os"
	"time"
)

var Debug = false

type AqiDataEntity struct {
	Id         int64
	City       string
	Aqi        int
	Time       int64
	Datasource int
	CrtDate    int64
	UpdDate    int64
	Version    int64
}

func (self *AqiDataEntity) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().Unix()
	self.CrtDate = now
	self.UpdDate = now
	return nil
}

func (self *AqiDataEntity) PreUpdate(s gorp.SqlExecutor) error {
	self.UpdDate = time.Now().Unix()
	return nil
}

//dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
type AqiDbHelper struct {
	dbConn *sql.DB
	dbmap  *gorp.DbMap
}

func NewAqiDbHelper(dbFile string) (*AqiDbHelper, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	aqiDbHelper := &AqiDbHelper{}
	aqiDbHelper.dbConn = db
	aqiDbHelper.dbmap = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	if Debug {
		aqiDbHelper.dbmap.TraceOn("", log.New(os.Stdout, "[gorpdebug]: ", log.LstdFlags))
	}
	initErr := aqiDbHelper.init()
	return aqiDbHelper, initErr
}

func (self *AqiDbHelper) init() error {
	aqiDataEntityTable := self.dbmap.AddTable(AqiDataEntity{}).SetKeys(true, "Id")
	aqiDataEntityTable.SetVersionCol("Version")
	err := self.dbmap.CreateTablesOpts(true)
	return err
}

func (self *AqiDbHelper) Close() error {
	return self.dbConn.Close()
}

func (self *AqiDbHelper) SaveAqiDataEntity(entity *AqiDataEntity) error {
	return self.dbmap.Insert(entity)
}

const (
	LastAqiEntitySql = `select a.Id, 
	                           a.City, 
							   a.Aqi, 
							   a.Time, 
							   a.Datasource, 
							   a.CrtDate, 
							   a.UpdDate, 
							   a.Version
		                  from AqiDataEntity a
		                 where a.City = ?
		                   and a.Time = (select max(b.Time) from AqiDataEntity b where b.City = ?)`
)

func (self *AqiDbHelper) GetLatestAqiEntity(city string) (*AqiDataEntity, error) {
	list, err := self.dbmap.Select(AqiDataEntity{}, LastAqiEntitySql, city, city)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list[0].(*AqiDataEntity), nil
	}
	return nil, nil
}
