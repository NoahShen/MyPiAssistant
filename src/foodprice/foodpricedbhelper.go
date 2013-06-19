package foodprice

import (
	"database/sql"
	_ "github.com/NoahShen/go-sqlite3"
	"github.com/NoahShen/gorp"
	"log"
	"os"
	"time"
)

type CityFoodPriceEntity struct {
	Id       int64
	City     string
	Time     int64
	Food     string
	Unit     string
	AvgPrice float64
	MaxPrice float64
	MaxSite  string
	MinPrice float64
	MinSite  string
	CrtDate  int64
	UpdDate  int64
	Version  int64
}

func (self *CityFoodPriceEntity) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().Unix()
	self.CrtDate = now
	self.UpdDate = now
	return nil
}

func (self *CityFoodPriceEntity) PreUpdate(s gorp.SqlExecutor) error {
	self.UpdDate = time.Now().Unix()
	return nil
}

type DistrictFoodPriceEntity struct {
	Id       int64
	District string
	Time     int64
	Food     string
	Price    float64
	Site     string
	CrtDate  int64
	UpdDate  int64
	Version  int64
}

func (self *DistrictFoodPriceEntity) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().Unix()
	self.CrtDate = now
	self.UpdDate = now
	return nil
}

func (self *DistrictFoodPriceEntity) PreUpdate(s gorp.SqlExecutor) error {
	self.UpdDate = time.Now().Unix()
	return nil
}

type FoodPriceDbHelper struct {
	dbConn *sql.DB
	dbmap  *gorp.DbMap
}

func NewFoodPriceDbHelper(dbFile string) (*FoodPriceDbHelper, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	foodPriceDbHelper := &FoodPriceDbHelper{}
	foodPriceDbHelper.dbConn = db
	foodPriceDbHelper.dbmap = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	if Debug {
		foodPriceDbHelper.dbmap.TraceOn("[gorp]", log.New(os.Stdout, "[pifoodprice]: ", log.LstdFlags))
	}
	initErr := foodPriceDbHelper.init()
	return foodPriceDbHelper, initErr
}

func (self *FoodPriceDbHelper) init() error {
	cityFoodPriceEntityTable := self.dbmap.AddTable(CityFoodPriceEntity{}).SetKeys(true, "Id")
	cityFoodPriceEntityTable.SetVersionCol("Version")
	districtFoodPriceEntityTable := self.dbmap.AddTable(DistrictFoodPriceEntity{}).SetKeys(true, "Id")
	districtFoodPriceEntityTable.SetVersionCol("Version")
	return self.dbmap.CreateTablesIfNotExists()
}

func (self *FoodPriceDbHelper) Close() error {
	return self.dbConn.Close()
}

func (self *FoodPriceDbHelper) AddCityFoodPrice(entity *CityFoodPriceEntity) error {
	return self.dbmap.Insert(entity)
}

func (self *FoodPriceDbHelper) AddDistrictFoodPrice(entity *DistrictFoodPriceEntity) error {
	return self.dbmap.Insert(entity)
}
