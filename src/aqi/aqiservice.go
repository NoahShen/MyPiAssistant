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
}

type config struct {
	DbFile        string `json:"dbFile,omitempty"`
	AqiPushCron   string `json:"aqiPushCron,omitempty"`
	AqiUpdateCron string `json:"aqiUpdateCron,omitempty"`
}

func (self *AqiService) GetServiceName() string {
	return "aqiService"
}

func (self *AqiService) Init(configRawMsg *json.RawMessage, pushCh chan<- *service.PushMessage) error {
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
	return nil
}

func (self *AqiService) Stop() error {
	self.cron.Stop()
	return self.dbHelper.Close()
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
	aqiDataEntity, err := self.dbHelper.GetLatestAqiEntity(cityEntity.CityName)
	if err != nil {
		return "", err
	}
	var aqiData *AqiData
	if aqiDataEntity != nil {
		now := time.Now().Unix()
		if (now - aqiDataEntity.Time) < 60*60*2 { // latest data within 2 hours
			aqiData = self.convertAqiDataEntityToAqiData(aqiDataEntity)
		}
	}

	if aqiData == nil { // no recent data in DB, get it from web
		aqi, err := FetchAqiFromWeb(cityEntity.CityName)
		if err != nil {
			return "", errors.New("获取数据失败！")
		}
		aqiData = aqi
	}
	return self.formatOutput(cityEntity.CityCNName, aqiData), nil
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
	cities, getCitiesErr := self.dbHelper.GetAllSubscribedCities()
	if getCitiesErr != nil {
		l4g.Error("Get subscribed cities error: %v", getCitiesErr)
		return
	}

	l4g.Debug("Updating aqi data, cities: %v", cities)

	for _, city := range cities {
		aqiData, fetchErr := FetchAqiFromWeb(city)
		if fetchErr != nil {
			l4g.Error("Fetch aqi data from web error: %v", fetchErr)
			continue
		}
		lastAqiDataEntity, getLatestErr := self.dbHelper.GetLatestAqiEntity(city)
		if getLatestErr != nil {
			l4g.Error("GetLatestAqiEntity error: %v", getLatestErr)
			continue
		}
		if aqiData.Time > lastAqiDataEntity.Time { // save new data
			entity := self.convertAqiDataToEntity(aqiData)
			saveError := self.dbHelper.SaveAqiDataEntity(entity)
			if saveError != nil {
				l4g.Error("Save AqiDataEntity error: %v", saveError)
				continue
			}
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
		lastAqiDataEntity, getLatestErr := self.dbHelper.GetLatestAqiEntity(userSubEntity.City)
		if getLatestErr != nil {
			l4g.Error("GetLatestAqiEntity error: %v", getLatestErr)
			continue
		}
		cityEntity := self.getCityEntity(userSubEntity.City)
		aqiData := self.convertAqiDataEntityToAqiData(lastAqiDataEntity)
		pushMsg := &service.PushMessage{}
		pushMsg.Type = service.Notification
		pushMsg.Username = userSubEntity.Username
		pushMsg.Message = self.formatOutput(cityEntity.CityCNName, aqiData)
		self.pushMsgChannel <- pushMsg
	}
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
	fTime := time.Unix(aqiData.Time, 0).Format("2006-01-02 15:04:05")
	ds := DatasourceMap[aqiData.Datasource]
	return fmt.Sprintf("%s的空气质量为%d, 发布时间%s, 数据来自%s", city, aqiData.Aqi, fTime, ds)
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
