package foodprice

import (
	"bytes"
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/robfig/cron"
	"service"
	"strings"
	"time"
)

type processFunc func(*FoodPriceService, string, []string) (string, error)

var commandHelp = map[string]string{
	"菜价":   "查询某个地区的菜价，如“菜价 青浦”，或“青浦菜价”",
	"订阅菜价": "订阅某个地区的菜价，如“订阅菜价 青浦”",
	"退订菜价": "退订某个地区的菜价，如“退订菜价 青浦”",
}

const (
	overTime = 60 * 60 * 24 * 2 // 2 days
)

type FoodPriceService struct {
	commandMap        map[string]processFunc
	aliasCommandMap   map[string]string
	config            *config
	pushMsgChannel    chan<- *service.PushMessage
	cron              *cron.Cron
	dbHelper          *FoodPriceDbHelper
	cityCNNameMap     map[string]string
	districtCNNameMap map[string]string
	started           bool
}

type config struct {
	DbFile          string `json:"dbFile,omitempty"`
	PricePushCron   string `json:"pricePushCron,omitempty"`
	PriceUpdateCron string `json:"priceUpdateCron,omitempty"`
}

func (self *FoodPriceService) GetServiceId() string {
	return "foodPriceService"
}

func (self *FoodPriceService) GetServiceName() string {
	return "菜价查询"
}

func (self *FoodPriceService) Init(configRawMsg *json.RawMessage, pushCh chan<- *service.PushMessage) error {
	self.started = false
	var c config
	err := json.Unmarshal(*configRawMsg, &c)
	if err != nil {
		return err
	}
	self.config = &c
	dbhelper, err := NewFoodPriceDbHelper(c.DbFile)
	if err != nil {
		return err
	}
	self.dbHelper = dbhelper
	l4g.Debug("Open food price DB successful: %s", c.DbFile)

	self.pushMsgChannel = pushCh
	self.cron = cron.New()
	self.cron.AddFunc(c.PricePushCron, func() {
		//self.pushAqiDataToUser()
	})
	self.cron.AddFunc(c.PriceUpdateCron, func() {
		self.updateFoodPriceData()
	})
	self.commandMap = map[string]processFunc{
		"foodprice": (*FoodPriceService).getFoodPrice,
		//"subprice":   (*FoodPriceService).subAqiData,
		//"unsubprice": (*FoodPriceService).unsubAqiData,
	}
	self.aliasCommandMap = map[string]string{
		"菜价":   "foodprice",
		"订阅菜价": "subprice",
		"退订菜价": "unsubprice",
	}

	self.cityCNNameMap = map[string]string{
		"上海": "shanghai",
	}
	self.districtCNNameMap = map[string]string{
		"浦东": "pudong",
		"黄浦": "huangpu",
		"徐汇": "xuhui",
		"长宁": "changning",
		"静安": "jingan",
		"普陀": "putuo",
		"闸北": "zhabei",
		"虹口": "hongkou",
		"杨浦": "yangpu",
		"宝山": "baoshan",
		"闵行": "minhang",
		"嘉定": "jiading",
		"金山": "jinshan",
		"松江": "songjiang",
		"青浦": "qingpu",
		"奉贤": "fengxian",
		"崇明": "chongming",
	}
	return nil
}

func (self *FoodPriceService) CommandFilter(command string, args []string) bool {
	if _, ok := self.aliasCommandMap[command]; ok {
		return true
	}

	if _, ok := self.commandMap[command]; ok {
		return true
	}
	return self.isStatmentCommand(command)
}

const (
	StatementCommandSuffix1 = "的菜价"
	StatementCommandSuffix2 = "菜价"
)

func (self *FoodPriceService) isStatmentCommand(command string) bool {
	return strings.HasSuffix(command, StatementCommandSuffix1) ||
		strings.HasSuffix(command, StatementCommandSuffix2)
}

func (self *FoodPriceService) parseStatementCommand(stmtCmd string, args []string) (string, []string) {
	if strings.HasSuffix(stmtCmd, StatementCommandSuffix1) {
		cityOrDistrict := stmtCmd[0:strings.Index(stmtCmd, StatementCommandSuffix1)]
		return "foodprice", []string{cityOrDistrict}
	} else if strings.HasSuffix(stmtCmd, StatementCommandSuffix2) {
		cityOrDistrict := stmtCmd[0:strings.Index(stmtCmd, StatementCommandSuffix2)]
		return "foodprice", []string{cityOrDistrict}
	}
	return stmtCmd, args
}

func (self *FoodPriceService) GetHelpMessage() string {
	var buffer bytes.Buffer
	for command, helpMsg := range commandHelp {
		buffer.WriteString(fmt.Sprintf("[%s]: %s\n", command, helpMsg))
	}
	return buffer.String()
}

func (self *FoodPriceService) StartService() error {
	self.cron.Start()
	self.started = true
	return nil
}

func (self *FoodPriceService) IsStarted() bool {
	return self.started
}

func (self *FoodPriceService) Stop() error {
	self.cron.Stop()
	err := self.dbHelper.Close()
	self.started = false
	return err
}

func (self *FoodPriceService) Handle(username, command string, args []string) (string, error) {
	realCommand := command
	realargs := args
	comm := self.aliasCommandMap[realCommand]
	if len(comm) > 0 {
		realCommand = comm
	}

	realCommand, realargs = self.parseStatementCommand(realCommand, realargs)
	l4g.Debug("realCommand, readargs: %s, %s", realCommand, realargs)
	f := self.commandMap[realCommand]
	if f == nil {
		return "", errors.New("命令错误！请输入\"help\"查询命令！")
	}
	return f(self, username, realargs)
}

func (self *FoodPriceService) getFoodPrice(username string, args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("缺少参数!")
	}
	cityOrDistrict := args[0]
	cityCode := self.getCityCode(cityOrDistrict)
	if len(cityCode) > 0 {
		foodPriceMessage, err := self.getCityFoodPrice(cityCode)
		if err != nil {
			l4g.Error("Get city food price failed: %v", err)
			return "", errors.New("获取数据失败！")
		}
		return foodPriceMessage, nil
	}

	districtCode := self.getDistrictCode(cityOrDistrict)
	if len(districtCode) > 0 {
		foodPriceMessage, err := self.getDistrictFoodPrice(districtCode)
		if err != nil {
			l4g.Error("Get district food price failed: %v", err)
			return "", errors.New("获取数据失败！")
		}
		return foodPriceMessage, nil
	}
	return "", errors.New("不支持该城市或地区的菜价查询！")
}

func (self *FoodPriceService) getCityCode(cityOrDistrict string) string {
	for cn, code := range self.cityCNNameMap {
		if cn == cityOrDistrict ||
			code == cityOrDistrict {
			return code
		}
	}
	return ""
}

func (self *FoodPriceService) getDistrictCode(cityOrDistrict string) string {
	for cn, code := range self.districtCNNameMap {
		if cn == cityOrDistrict ||
			code == cityOrDistrict {
			return code
		}
	}
	return ""
}

var cityFoodPriceMsgCache = map[string]string{}

func (self *FoodPriceService) getCityFoodPrice(city string) (string, error) {
	foodPriceMsg, hitCache := cityFoodPriceMsgCache[city]
	if hitCache {
		l4g.Debug("Hit cityFoodPriceMsgCache, city: %s", city)
		return foodPriceMsg, nil
	}
	var cityFoodPrices []*CityFoodPrice
	entities, err := self.dbHelper.GetLatestCityFoodPriceEntity(city, int64(overTime))
	if err != nil {
		return "", err
	}

	if len(entities) > 0 {
		cityFoodPrices = self.convertCityEntityToFoodPrice(entities)
	}
	if len(cityFoodPrices) == 0 {
		var e error
		l4g.Debug("get %s food price from web", city)
		cityFoodPrices, e = FetchCityFoodPrice(city)
		if e != nil {
			return "", e
		}
	}
	if len(cityFoodPrices) == 0 {
		return "无记录", nil
	}
	msg := self.formatCityFoodPrice(cityFoodPrices)
	url, _ := GetCityFoodPriceUrl(city)
	msg = msg + fmt.Sprintf("\n详细信息请点击：%s", url)
	cityFoodPriceMsgCache[city] = msg
	return msg, nil
}

func (self *FoodPriceService) convertCityEntityToFoodPrice(entities []*CityFoodPriceEntity) []*CityFoodPrice {
	cityFoodPrices := make([]*CityFoodPrice, 0)
	for _, e := range entities {
		cityFoodPrice := &CityFoodPrice{}
		cityFoodPrice.City = e.City
		cityFoodPrice.Time = e.Time
		cityFoodPrice.Food = e.Food
		cityFoodPrice.Unit = e.Unit
		cityFoodPrice.AvgPrice = e.AvgPrice
		cityFoodPrice.MaxPrice = e.MaxPrice
		cityFoodPrice.MaxSite = e.MaxSite
		cityFoodPrice.MinPrice = e.MinPrice
		cityFoodPrice.MinSite = e.MinSite
		cityFoodPrices = append(cityFoodPrices, cityFoodPrice)
	}
	return cityFoodPrices
}

func (self *FoodPriceService) formatCityFoodPrice(cityFoodPrices []*CityFoodPrice) string {
	var buffer bytes.Buffer
	for _, p := range cityFoodPrices {
		buffer.WriteString(fmt.Sprintf("\n%s均价: %.2f", p.Food, p.AvgPrice))
	}
	fTime := time.Unix(cityFoodPrices[0].Time, 0).Format("2006-01-02 15:04")
	buffer.WriteString(fmt.Sprintf("\n采集时间：%s", fTime))
	return buffer.String()
}

var districtFoodPriceMsgCache = map[string]string{}

func (self *FoodPriceService) getDistrictFoodPrice(district string) (string, error) {
	foodPriceMsg, hitCache := districtFoodPriceMsgCache[district]
	if hitCache {
		l4g.Debug("Hit districtFoodPriceMsgCache, district: %s", district)
		return foodPriceMsg, nil
	}
	var districtFoodPrices []*DistrictFoodPrice
	entities, err := self.dbHelper.GetLatestDistrictFoodPriceEntity(district, int64(overTime))
	if err != nil {
		return "", err
	}
	if len(entities) > 0 {
		districtFoodPrices = self.convertDistrictEntityToFoodPrice(entities)
	}
	if len(districtFoodPrices) == 0 {
		var e error
		l4g.Debug("get %s food price from web", district)
		districtFoodPrices, e = FetchDistrictFoodPrice(district)
		if e != nil {
			return "", e
		}
	}
	if len(districtFoodPrices) == 0 {
		return "无记录", nil
	}
	msg := self.formatDistrictFoodPrice(districtFoodPrices)
	url, _ := GetDistrictFoodPriceUrl(district)
	msg = msg + fmt.Sprintf("\n详细信息请点击：%s", url)
	districtFoodPriceMsgCache[district] = msg
	return msg, nil
}

func (self *FoodPriceService) convertDistrictEntityToFoodPrice(entities []*DistrictFoodPriceEntity) []*DistrictFoodPrice {
	districtFoodPriceMap := make(map[string]*DistrictFoodPrice, 0)
	for _, e := range entities {
		districtFoodPrice := districtFoodPriceMap[e.Food]
		if districtFoodPrice == nil {
			districtFoodPrice = &DistrictFoodPrice{}
			districtFoodPrice.District = e.District
			districtFoodPrice.Time = e.Time
			districtFoodPrice.Food = e.Food
			districtFoodPriceMap[e.Food] = districtFoodPrice
		}
		priceSite := &PriceSite{}
		priceSite.Price = e.Price
		priceSite.Site = e.Site
		districtFoodPrice.PricesSites = append(districtFoodPrice.PricesSites, *priceSite)
	}
	districtFoodPrices := make([]*DistrictFoodPrice, 0)
	for _, fp := range districtFoodPriceMap {
		districtFoodPrices = append(districtFoodPrices, fp)
	}
	return districtFoodPrices
}

func (self *FoodPriceService) formatDistrictFoodPrice(districtFoodPrices []*DistrictFoodPrice) string {
	var buffer bytes.Buffer
	for _, p := range districtFoodPrices {
		var avgPrice float64 = 0.0
		count := 0
		for _, ps := range p.PricesSites {
			if ps.Price > 0 { // invalid price
				avgPrice += ps.Price
				count++
			}
		}
		buffer.WriteString(fmt.Sprintf("\n%s的均价：%.2f", p.Food, avgPrice/float64(count)))
	}
	fTime := time.Unix(districtFoodPrices[0].Time, 0).Format("2006-01-02 15:04")
	buffer.WriteString(fmt.Sprintf("\n采集时间：%s", fTime))
	return buffer.String()
}

func (self *FoodPriceService) updateFoodPriceData() {
	for cityCode, _ := range CityMap {
		lastUpdateTime, timeErr := self.dbHelper.GetLastUpdateCityPriceTime(cityCode)
		if timeErr != nil {
			l4g.Error("get %s lastUpdateTime error:%v", cityCode, timeErr)
			continue
		}
		cityFoodPrices, err := FetchCityFoodPrice(cityCode)
		if err != nil {
			l4g.Error("get %s food price from web error:%v", cityCode, err)
			continue
		}
		if len(cityFoodPrices) == 0 {
			l4g.Error("get %s food price from web is empty", cityCode)
			continue
		}
		if cityFoodPrices[0].Time > lastUpdateTime {
			for _, cityFoodPrice := range cityFoodPrices {
				self.dbHelper.AddCityFoodPrice(self.convertCityFoodPriceToEntity(cityFoodPrice))
			}
			delete(cityFoodPriceMsgCache, cityCode)
			l4g.Debug("update %s food price success.", cityCode)
		}
	}

	for districtCode, _ := range DistrictMap {
		lastUpdateTime, timeErr := self.dbHelper.GetLastUpdateDistrictPriceTime(districtCode)
		if timeErr != nil {
			l4g.Error("get %s lastUpdateTime error:%v", districtCode, timeErr)
			continue
		}
		districtFoodPrices, err := FetchDistrictFoodPrice(districtCode)
		if err != nil {
			l4g.Error("get %s food price from web error:%v", districtCode, err)
			continue
		}
		if len(districtFoodPrices) == 0 {
			l4g.Error("get %s food price from web is empty", districtCode)
			continue
		}
		if districtFoodPrices[0].Time > lastUpdateTime {
			for _, districtFoodPrice := range districtFoodPrices {
				entities := self.convertDistrictFoodPriceToEntity(districtFoodPrice)
				for _, e := range entities {
					self.dbHelper.AddDistrictFoodPrice(e)
				}
			}
			delete(districtFoodPriceMsgCache, districtCode)
			l4g.Debug("update %s food price success.", districtCode)
		}
	}
}

func (self *FoodPriceService) convertCityFoodPriceToEntity(foodPrice *CityFoodPrice) *CityFoodPriceEntity {
	entity := &CityFoodPriceEntity{}
	entity.City = foodPrice.City
	entity.Time = foodPrice.Time
	entity.Food = foodPrice.Food
	entity.Unit = foodPrice.Unit
	entity.AvgPrice = foodPrice.AvgPrice
	entity.MaxPrice = foodPrice.MaxPrice
	entity.MaxSite = foodPrice.MaxSite
	entity.MinPrice = foodPrice.MinPrice
	entity.MinSite = foodPrice.MinSite
	return entity
}

func (self *FoodPriceService) convertDistrictFoodPriceToEntity(foodPrice *DistrictFoodPrice) []*DistrictFoodPriceEntity {
	entities := make([]*DistrictFoodPriceEntity, 0)
	for _, p := range foodPrice.PricesSites {
		entity := &DistrictFoodPriceEntity{}
		entity.District = foodPrice.District
		entity.Time = foodPrice.Time
		entity.Food = foodPrice.Food
		entity.Price = p.Price
		entity.Site = p.Site
		entities = append(entities, entity)
	}
	return entities
}
