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

// Usersub
type UserSubEntity struct {
	Id             int64
	Username       string
	CityOrDistrict string
	SubStatus      int //0 for unsub, 1 for sub
	CrtDate        int64
	UpdDate        int64
	Version        int64
}

func (self *UserSubEntity) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().Unix()
	self.CrtDate = now
	self.UpdDate = now
	return nil
}

func (self *UserSubEntity) PreUpdate(s gorp.SqlExecutor) error {
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
	userSubEntityTable := self.dbmap.AddTable(UserSubEntity{}).SetKeys(true, "Id")
	userSubEntityTable.SetVersionCol("Version")

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
	LastCityFoodPriceEntitySql = `
    select a.Id, 
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
       and a.Time = (select max(b.Time) from cityfoodpriceentity b where b.City = ?)
       and a.Time > strftime('%s','now') - ?`
)

func (self *FoodPriceDbHelper) GetLatestCityFoodPriceEntity(city string, latestTime int64) ([]*CityFoodPriceEntity, error) {
	list, err := self.dbmap.Select(CityFoodPriceEntity{}, LastCityFoodPriceEntitySql, city, city, latestTime)
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
	LastUpdateCityPriceTimeSql = `
    select ifnull(max(b.Time), 0) from cityfoodpriceentity b where b.City = ?`
)

func (self *FoodPriceDbHelper) GetLastUpdateCityPriceTime(city string) (int64, error) {
	time, err := self.dbmap.SelectInt(LastUpdateCityPriceTimeSql, city)
	return time, err
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
       and a.Time = (select max(b.Time) from districtfoodpriceentity b where b.District = ?)
	   and a.Time > strftime('%s','now') - ?`
)

func (self *FoodPriceDbHelper) GetLatestDistrictFoodPriceEntity(district string, latestTime int64) ([]*DistrictFoodPriceEntity, error) {
	list, err := self.dbmap.Select(DistrictFoodPriceEntity{}, LastDistrictFoodPriceEntitySql, district, district, latestTime)
	if err != nil {
		return nil, err
	}
	entities := make([]*DistrictFoodPriceEntity, len(list))
	for i, item := range list {
		entities[i] = item.(*DistrictFoodPriceEntity)
	}
	return entities, nil
}

const (
	LastUpdateDistrictPriceTimeSql = `
    select ifnull(max(b.Time), 0) from districtfoodpriceentity b where b.District = ?`
)

func (self *FoodPriceDbHelper) GetLastUpdateDistrictPriceTime(district string) (int64, error) {
	time, err := self.dbmap.SelectInt(LastUpdateDistrictPriceTimeSql, district)
	return time, err
}

func (self *FoodPriceDbHelper) AddUserSub(userSub *UserSubEntity) error {
	return self.dbmap.Insert(userSub)
}

func (self *FoodPriceDbHelper) UpdateUserSub(userSub *UserSubEntity) error {
	_, err := self.dbmap.Update(userSub)
	return err
}

const (
	GetUserSubSql = `select u.Id, 
                        	u.Username, 
							u.CityOrDistrict, 
						    u.SubStatus, 
						    u.CrtDate, 
						    u.UpdDate, 
						    u.Version
	                   from UserSubEntity u
	                  where u.Username = ?
	                    and u.CityOrDistrict = ?`
)

func (self *FoodPriceDbHelper) GetUserSub(username, city string) (*UserSubEntity, error) {
	list, err := self.dbmap.Select(UserSubEntity{}, GetUserSubSql, username, city)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list[0].(*UserSubEntity), nil
	}
	return nil, nil
}

const (
	GetSubscribedUserSql = `select u.Id, 
                        	       u.Username, 
								   u.CityOrDistrict, 
						           u.SubStatus, 
						           u.CrtDate, 
						           u.UpdDate, 
						           u.Version
	                          from UserSubEntity u
	                         where u.SubStatus = ?`
)

func (self *FoodPriceDbHelper) GetSubscribedUser() ([]*UserSubEntity, error) {
	list, err := self.dbmap.Select(UserSubEntity{}, GetSubscribedUserSql, 1)
	if err != nil {
		return nil, err
	}
	entities := make([]*UserSubEntity, len(list))
	for i, item := range list {
		entities[i] = item.(*UserSubEntity)
	}
	return entities, nil
}
