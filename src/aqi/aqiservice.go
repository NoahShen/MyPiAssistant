package aqi

import (
	"bytes"
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/robfig/cron"
	"service"
	"time"
)

type processFunc func(*AqiService, string, []string) (string, error)

var commandHelp = map[string]string{}

type AqiService struct {
	commandMap      map[string]processFunc
	aliasCommandMap map[string]string
	config          *config
	pushMsgChannel  chan<- *service.PushMessage
	cron            *cron.Cron
	dbHelper        *AqiDbHelper
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
	//self.cron.AddFunc(c.AqiPushCron, func() {
	//	self.updateDownloadStat()
	//})
	//self.cron.AddFunc(c.AqiUpdateCron, func() {
	//	self.updateDownloadStat()
	//})
	self.commandMap = map[string]processFunc{
		"currentAqi": (*AqiService).getCurrentAqi,
	}
	self.aliasCommandMap = map[string]string{
		"空气质量": "currentAqi",
	}

	return nil
}

func (self *AqiService) CommandFilter(command string, args []string) bool {
	if _, ok := self.aliasCommandMap[command]; ok {
		return true
	}

	if _, ok := self.commandMap[command]; ok {
		return true
	}
	return false
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
	comm := self.aliasCommandMap[command]
	if comm == "" || len(comm) == 0 {
		comm = command
	}

	f := self.commandMap[comm]
	if f == nil {
		return "", errors.New("命令错误！请输入\"help\"查询命令！")
	}
	return f(self, username, args)
}

func (self *AqiService) getCurrentAqi(username string, args []string) (string, error) {
	if args == nil || len(args) == 0 {
		return "", errors.New("缺少参数!")
	}
	city := args[0]
	aqiDataEntity, err := self.dbHelper.GetLatestAqiEntity(city)
	if err != nil {
		return "", err
	}
	var aqiData *AqiData
	if aqiDataEntity != nil {
		aqiData = self.convertAqiDataEntityToAqiData(aqiDataEntity)
	} else {
		aqi, err := FetchAqiFromWeb(city)
		if err != nil {
			return "", err
		}
		aqiData = aqi
	}
	return self.formatOutput(aqiData), nil
}

func (self *AqiService) convertAqiDataEntityToAqiData(entity *AqiDataEntity) *AqiData {
	aqiData := &AqiData{}
	aqiData.Aqi = entity.Aqi
	aqiData.Time = entity.Time
	aqiData.Datasource = DataSource(entity.Datasource)
	return aqiData
}

func (self *AqiService) formatOutput(aqiData *AqiData) string {
	fTime := time.Unix(aqiData.Time, 0).Format("2006-01-02 15:04:05")
	ds := ""
	if aqiData.Datasource == USEmbassy {
		ds = "美国大使馆"
	} else {
		ds = "中国官方"
	}
	return fmt.Sprintf("空气质量为%d, 发布时间%s, 数据来自%s", aqiData.Aqi, fTime, ds)
}
