package aqi

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

type processFunc func(*AqiService, string, []string) (string, error)

var commandHelp = map[string]string{
	"空气质量":   "查询某个城市的空气质量，如“空气质量 上海”，或“上海的空气质量”",
	"订阅空气质量": "订阅某个城市的空气质量，如“订阅空气质量 上海”",
	"退订空气质量": "退订某个城市的空气质量，如“退订空气质量 上海”",
}

type AqiService struct {
	commandMap      map[string]processFunc
	aliasCommandMap map[string]string
	config          *config
	pushMsgChannel  chan<- *service.PushMessage
	cron            *cron.Cron
	dbHelper        *AqiDbHelper
	cityNameMap     map[string]*AqiCityEntity
	cityCNNameMap   map[string]*AqiCityEntity
	started         bool
}

type config struct {
	DbFile        string `json:"dbFile,omitempty"`
	AqiPushCron   string `json:"aqiPushCron,omitempty"`
	AqiUpdateCron string `json:"aqiUpdateCron,omitempty"`
	LatestHour    int    `json:"latestHour,omitempty"`
}

func (self *AqiService) GetServiceId() string {
	return "aqiService"
}

func (self *AqiService) GetServiceName() string {
	return "空气质量查询"
}

func (self *AqiService) Init(configRawMsg *json.RawMessage, pushCh chan<- *service.PushMessage) error {
	self.started = false
	var c config
	err := json.Unmarshal(*configRawMsg, &c)
	if err != nil {
		return err
	}
	self.config = &c
	aqiDbhelper, err := NewAqiDbHelper(c.DbFile)
	if err != nil {
		return err
	}
	self.dbHelper = aqiDbhelper
	l4g.Debug("Open aqi DB successful: %s", c.DbFile)

	self.pushMsgChannel = pushCh
	self.cron = cron.New()
	self.cron.AddFunc(c.AqiPushCron, func() {
		self.pushAqiDataToUser()
	})
	self.cron.AddFunc(c.AqiUpdateCron, func() {
		self.updateAqiData()
	})
	self.commandMap = map[string]processFunc{
		"currentaqi":   (*AqiService).getCurrentAqi,
		"subaqidata":   (*AqiService).subAqiData,
		"unsubaqidata": (*AqiService).unsubAqiData,
	}
	self.aliasCommandMap = map[string]string{
		"空气质量":   "currentaqi",
		"订阅空气质量": "subaqidata",
		"退订空气质量": "unsubaqidata",
	}

	entities, getCitiesErr := self.dbHelper.GetAllCities()
	if getCitiesErr != nil {
		return getCitiesErr
	}
	self.cityNameMap = make(map[string]*AqiCityEntity)
	self.cityCNNameMap = make(map[string]*AqiCityEntity)
	for _, entity := range entities {
		self.cityNameMap[entity.CityName] = entity
		self.cityCNNameMap[entity.CityCNName] = entity
	}
	l4g.Debug("init cityMap successful, cities number: %d", len(self.cityNameMap))
	return nil
}

func (self *AqiService) CommandFilter(command string, args []string) bool {
	if _, ok := self.aliasCommandMap[command]; ok {
		return true
	}

	if _, ok := self.commandMap[command]; ok {
		return true
	}
	return self.isStatmentCommand(command)
}

const (
	StatementCommandSuffix1 = "的空气质量"
	StatementCommandSuffix2 = "空气质量"
)

func (self *AqiService) isStatmentCommand(command string) bool {
	return strings.HasSuffix(command, StatementCommandSuffix1) ||
		strings.HasSuffix(command, StatementCommandSuffix2)
}

func (self *AqiService) parseStatementCommand(stmtCmd string, args []string) (string, []string) {
	if strings.HasSuffix(stmtCmd, StatementCommandSuffix1) {
		city := stmtCmd[0:strings.Index(stmtCmd, StatementCommandSuffix1)]
		return "currentaqi", []string{city}
	} else if strings.HasSuffix(stmtCmd, StatementCommandSuffix2) {
		city := stmtCmd[0:strings.Index(stmtCmd, StatementCommandSuffix2)]
		return "currentaqi", []string{city}
	}
	return stmtCmd, args
}

func (self *AqiService) GetHelpMessage() string {
	var buffer bytes.Buffer
	for command, helpMsg := range commandHelp {
		buffer.WriteString(fmt.Sprintf("[%s]: %s\n", command, helpMsg))
	}
	return buffer.String()
}

func (self *AqiService) StartService() error {
	self.cron.Start()
	self.started = true
	return nil
}

func (self *AqiService) IsStarted() bool {
	return self.started
}

func (self *AqiService) Stop() error {
	self.cron.Stop()
	err := self.dbHelper.Close()
	self.started = false
	return err
}

func (self *AqiService) Handle(username, command string, args []string) (string, error) {
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

func (self *AqiService) getCurrentAqi(username string, args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("缺少参数!")
	}
	city := args[0]
	cityEntity := self.getCityEntity(strings.ToLower(city))
	if cityEntity == nil {
		return "", errors.New("不支持该城市的空气质量查询！")
	}
	statisticsMessage := self.getStatisticsAqiDataMessage(cityEntity.CityName)
	if len(statisticsMessage) > 0 {
		return statisticsMessage, nil
	}
	aqiData, err := FetchAqiFromWeb(cityEntity.CityName)
	if err != nil {
		return "", errors.New("获取数据失败！")
	}
	return self.formatOutput(cityEntity.CityCNName, aqiData[0]), nil
}

func (self *AqiService) subAqiData(username string, args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("缺少参数!")
	}
	city := args[0]
	cityEntity := self.getCityEntity(strings.ToLower(city))
	if cityEntity == nil {
		return "", errors.New("不支持该城市的空气质量查询！")
	}
	userSubEntity, err := self.dbHelper.GetUserSub(username, cityEntity.CityName)
	if err != nil {
		l4g.Error("GetUserSub error in Subscribe aqi: username: %s, error: %v", username, err)
		return "", errors.New("订阅失败！")
	}
	if userSubEntity == nil {
		userSubEntity = &UserSubEntity{}
		userSubEntity.Username = username
		userSubEntity.City = cityEntity.CityName
		userSubEntity.SubStatus = 1
		addError := self.dbHelper.AddUserSub(userSubEntity)
		if addError != nil {
			l4g.Error("AddUserSub error in Subscribe aqi: username: %s, error: %v", username, err)
			return "", errors.New("订阅失败！")
		}
	} else {
		userSubEntity.SubStatus = 1
		updateError := self.dbHelper.UpdateUserSub(userSubEntity)
		if updateError != nil {
			l4g.Error("UpdateUserSub error in Subscribe aqi: username: %s, error: %v", username, err)
			return "", errors.New("订阅失败！")
		}
	}
	return "订阅成功！", nil
}

func (self *AqiService) unsubAqiData(username string, args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("缺少参数!")
	}
	city := args[0]
	cityEntity := self.getCityEntity(strings.ToLower(city))
	if cityEntity == nil {
		return "", errors.New("不支持该城市的空气质量查询！")
	}
	userSubEntity, err := self.dbHelper.GetUserSub(username, cityEntity.CityName)
	if err != nil {
		l4g.Error("GetUserSub error in Unsubscribe aqi: username: %s, error: %v", username, err)
		return "", errors.New("退订失败！")
	}
	if userSubEntity != nil {
		userSubEntity.SubStatus = 0
		updateError := self.dbHelper.UpdateUserSub(userSubEntity)
		if updateError != nil {
			l4g.Error("UpdateUserSub error in Unsubscribe aqi: username: %s, error: %v", username, err)
			return "", errors.New("退订失败！")
		}
	} else {
		return "未订阅过该城市的空气质量信息！", nil
	}
	return "退订成功！", nil
}

func (self *AqiService) updateAqiData() {
	cities, getCitiesErr := self.dbHelper.GetNeedUpdateCities(60 * 60) // update one time in one hour
	if getCitiesErr != nil {
		l4g.Error("Get need update cities error: %v", getCitiesErr)
		return
	}

	l4g.Debug("Need updating cities: %v", cities)

	for _, cityInfo := range cities {
		aqiData, fetchErr := FetchAqiFromWeb(cityInfo.City)
		if fetchErr != nil {
			l4g.Error("Fetch aqi data from web error: %v", fetchErr)
			continue
		}
		removeCache := false
		for _, aqi := range aqiData {
			if aqi.Time > cityInfo.LatestUpdateTime &&
				aqi.Aqi >= 0 { // filter invalied aqi data
				// save new data
				entity := self.convertAqiDataToEntity(aqi)
				saveError := self.dbHelper.SaveAqiDataEntity(entity)
				if saveError != nil {
					l4g.Error("Save AqiDataEntity error: %v", saveError)
					continue
				}
				removeCache = true
			}
		}
		if removeCache {
			delete(aqiStatisticsCache, cityInfo.City)
			l4g.Debug("Remove %s statistics cache.", cityInfo.City)
		}

	}
}

func (self *AqiService) pushAqiDataToUser() {
	userSubEntities, getSubUserError := self.dbHelper.GetSubscribedUser()
	if getSubUserError != nil {
		l4g.Error("Get subscribed user error: %v", getSubUserError)
		return
	}

	for _, userSubEntity := range userSubEntities {
		statisticsMessage := self.getStatisticsAqiDataMessage(userSubEntity.City)
		if len(statisticsMessage) == 0 {
			continue
		}
		pushMsg := &service.PushMessage{}
		pushMsg.Type = service.Notification
		pushMsg.Username = userSubEntity.Username
		pushMsg.Message = statisticsMessage
		self.pushMsgChannel <- pushMsg
	}
}

var aqiStatisticsCache = map[string]string{}

func (self *AqiService) getStatisticsAqiDataMessage(city string) string {
	statisticsMessage := aqiStatisticsCache[city]
	if len(statisticsMessage) == 0 {
		//get the data of the latest several hours
		t := time.Now().Add(time.Duration(-self.config.LatestHour) * time.Hour).Unix()
		latestAqiDataEntities, getLatestErr := self.dbHelper.GetAqiDataAfterTime(city, t)
		if getLatestErr != nil {
			l4g.Error("GetLatestAqiEntity error: %v", getLatestErr)
			return ""
		}
		if len(latestAqiDataEntities) == 0 {
			l4g.Info("None recent aqi data, city: %s", city)
			return ""
		}

		averageAqi, maxEntities, minEntities := self.calStatisticsAqiData(latestAqiDataEntities)

		cityEntity := self.getCityEntity(city)
		aqiData := self.convertAqiDataEntityToAqiData(latestAqiDataEntities[0])
		statisticsMessage = self.formatStatisticsOutput(cityEntity.CityCNName, aqiData, self.config.LatestHour,
			averageAqi, maxEntities, minEntities)
		aqiStatisticsCache[city] = statisticsMessage
	} else {
		l4g.Debug("Hit statistics cache, city: %s", city)
	}
	return statisticsMessage
}

func (self *AqiService) formatStatisticsOutput(cityName string, latestAqi *AqiData, latestHour, avgAqi int, maxEntities, minEntities []*AqiDataEntity) string {

	var buffer bytes.Buffer
	fTime := time.Unix(latestAqi.Time, 0).Format("15:04")
	ds := DatasourceMap[latestAqi.Datasource]
	buffer.WriteString(fmt.Sprintf("%s的空气指数为%d (%s)", fTime, latestAqi.Aqi, self.getAqiLevel(latestAqi.Aqi)))
	buffer.WriteString(fmt.Sprintf("\n-------"))
	buffer.WriteString(fmt.Sprintf("\n%s最近%d小时的平均指数为%d", cityName, latestHour, avgAqi))
	if len(maxEntities) > 0 {
		buffer.WriteString("\n在")
		for i, e := range maxEntities {
			if i != 0 {
				buffer.WriteString(",")
			}
			buffer.WriteString(time.Unix(e.Time, 0).Format("15:04"))
		}
		buffer.WriteString(fmt.Sprintf("最高，为:%d", maxEntities[0].Aqi))
	}

	if len(minEntities) > 0 {
		buffer.WriteString("\n在")
		for i, e := range minEntities {
			if i != 0 {
				buffer.WriteString(",")
			}
			buffer.WriteString(time.Unix(e.Time, 0).Format("15:04"))
		}
		buffer.WriteString(fmt.Sprintf("最低，为:%d", minEntities[0].Aqi))
	}
	buffer.WriteString(fmt.Sprintf("\n数据来自%s", ds))
	return buffer.String()

}

// calculate statistics aqi data, max aqi and min aqi may be more than one entity
func (self *AqiService) calStatisticsAqiData(aqiDataEntities []*AqiDataEntity) (int, []*AqiDataEntity, []*AqiDataEntity) {
	maxAqiEntities := make([]*AqiDataEntity, 0)
	minAqiEntities := make([]*AqiDataEntity, 0)
	aqiSum := 0
	for _, entity := range aqiDataEntities {
		aqiSum += entity.Aqi
		if len(maxAqiEntities) == 0 ||
			maxAqiEntities[0].Aqi < entity.Aqi {
			maxAqiEntities = []*AqiDataEntity{entity}
		} else if maxAqiEntities[0].Aqi == entity.Aqi {
			maxAqiEntities = append(maxAqiEntities, entity)
		}

		if len(minAqiEntities) == 0 ||
			minAqiEntities[0].Aqi > entity.Aqi {
			minAqiEntities = []*AqiDataEntity{entity}
		} else if minAqiEntities[0].Aqi == entity.Aqi {
			minAqiEntities = append(minAqiEntities, entity)
		}
	}
	return aqiSum / len(aqiDataEntities), maxAqiEntities, minAqiEntities
}

func (self *AqiService) convertAqiDataToEntity(aqiData *AqiData) *AqiDataEntity {
	entity := &AqiDataEntity{}
	entity.City = aqiData.City
	entity.Aqi = aqiData.Aqi
	entity.Time = aqiData.Time
	entity.Datasource = int(aqiData.Datasource)
	return entity
}

func (self *AqiService) convertAqiDataEntityToAqiData(entity *AqiDataEntity) *AqiData {
	aqiData := &AqiData{}
	aqiData.Aqi = entity.Aqi
	aqiData.Time = entity.Time
	aqiData.Datasource = DataSource(entity.Datasource)
	return aqiData
}

func (self *AqiService) formatOutput(city string, aqiData *AqiData) string {
	fTime := time.Unix(aqiData.Time, 0).Format("2006-01-02 15:04")
	ds := DatasourceMap[aqiData.Datasource]
	return fmt.Sprintf("%s空气质量指数为%d (%s)\n更新时间：%s\n数据来自%s", city, aqiData.Aqi, self.getAqiLevel(aqiData.Aqi), fTime, ds)
}

func (self *AqiService) getCityEntity(city string) *AqiCityEntity {
	if entity, ok := self.cityCNNameMap[city]; ok {
		return entity
	}

	if entity, ok := self.cityNameMap[city]; ok {
		return entity
	}
	return nil
}

func (self *AqiService) getAqiLevel(aqi int) string {
	switch {
	case aqi <= 50:
		return "一级：优"
	case 50 < aqi && aqi <= 100:
		return "二级：良"
	case 100 < aqi && aqi <= 150:
		return "三级：轻度污染"
	case 150 < aqi && aqi <= 200:
		return "四级：中度污染"
	case 200 < aqi && aqi <= 300:
		return "五级：重度污染"
	}
	// > 300
	return "六级：严重污染"
}
