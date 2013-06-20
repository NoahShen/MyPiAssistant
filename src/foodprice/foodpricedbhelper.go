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

const (
	LastCityFoodPriceEntitySql = `select a.Id, 
	                           a.City, 
							   a.Time, 
							   a.Food, 
							   a.Unit, 
							   a.AvgPrice, 
							   a.MaxPrice,
							   a.MaxSite,
							   a.MinPrice,
							   a.MinSite,
							   a.CrtDate, 
							   a.UpdDate, 
							   a.Version
		                  from cityfoodpriceentity a
		                 where a.City = ?
		                   and a.Time = (select max(b.Time) from cityfoodpriceentity b where b.City = ?)`
)

func (self *FoodPriceDbHelper) GetLatestCityFoodPriceEntity(city string) ([]*CityFoodPriceEntity, error) {
	list, err := self.dbmap.Select(CityFoodPriceEntity{}, LastCityFoodPriceEntitySql, city, city)
	if err != nil {
		return nil, err
	}
	entities := make([]*CityFoodPriceEntity, len(list))
	for i, item := range list {
		entities[i] = item.(*CityFoodPriceEntity)
	}
	return entities, nil
}

const (
	LastDistrictFoodPriceEntitySql = `
    select a.Id, 
           a.District, 
           a.Time, 
           a.Food, 
           a.Price, 
           a.Site, 
           a.CrtDate, 
           a.UpdDate, 
           a.Version
      from districtfoodpriceentity a
     where a.District = ?
       and a.Time = (select max(b.Time) from districtfoodpriceentity b where b.District = ?)`
)

func (self *FoodPriceDbHelper) GetLatestDistrictFoodPriceEntity(district string) ([]*DistrictFoodPriceEntity, error) {
	list, err := self.dbmap.Select(DistrictFoodPriceEntity{}, LastDistrictFoodPriceEntitySql, district, district)
	if err != nil {
		return nil, err
	}
	entities := make([]*DistrictFoodPriceEntity, len(list))
	for i, item := range list {
		entities[i] = item.(*DistrictFoodPriceEntity)
	}
	return entities, nil
}
