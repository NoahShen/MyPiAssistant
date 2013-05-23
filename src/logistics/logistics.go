package logistics

import (
	"bytes"
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/robfig/cron"
	"service"
	"sort"
	"strconv"
	"time"
)

const (
	LOGISTICS_UPDATE_TIMEOUT = 1 * 24 * 60 * 60
)

var companyMap = map[string]string{
	"申通":  "shentong",
	"EMS": "ems",
	"顺丰":  "shunfeng",
	"圆通":  "yuantong",
	"中通":  "zhongtong",
	"如风达": "rufengda",
	"韵达":  "yunda",
	"天天":  "tiantian",
	"汇通":  "huitongkuaidi",
	"全峰":  "quanfengkuaidi",
	"德邦":  "debangwuliu",
	"宅急送": "zhaijisong",
}

type byTime []LogisticsRecordEntity

func (s byTime) Len() int {
	return len(s)
}

func (s byTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byTime) Less(i, j int) bool {
	return s[i].Time < s[j].Time
}

type processFunc func(*LogisticsService, string, []string) (string, error)

var commandHelp = map[string]string{
	"跟踪快递": "订阅某条快递信息，命令格式:跟踪快递 快递名称 物流公司(或代码) 物流单号",
	"取消快递": "取消订阅某条快递, 命令格式:取消快递 物流公司(或代码) 物流单号 或 取消快递 快递名",
	"查询快递": "查询某条快递配送进度，在查询前不需要添加跟踪该快递，该物流单号命令格式: 查询快递 物流公司(或代码) 物流单号 或 查询快递 快递名",
	"快递记录": "查询最近完成的物流单",
	"我的快递": "查询已跟踪订阅的所有在途快递",
	"物流公司": "查询支持的物流公司",
}

type logisticsTrackingInfo struct {
	Username      string
	LogisticsName string
	NewRecords    []LogisticsRecordEntity
}

type config struct {
	DbFile              string `json:"dbFile,omitempty"`
	BeforeLastUpdate    int64  `json:"beforeLastUpdate,omitempty"`
	LogisticsUpdateCron string `json:"logisticsUpdateCron,omitempty"`
}

type LogisticsService struct {
	logisticsdb     *LogisticsDb
	commandMap      map[string]processFunc
	aliasCommandMap map[string]string
	config          *config
	pushMsgChannel  chan<- *service.PushMessage
	cron            *cron.Cron
	started         bool
}

func (self *LogisticsService) GetServiceId() string {
	return "logisticsquery"
}

func (self *LogisticsService) GetServiceName() string {
	return "物流查询"
}

func (self *LogisticsService) Init(configRawMsg *json.RawMessage, pushCh chan<- *service.PushMessage) error {
	self.started = false
	var c config
	err := json.Unmarshal(*configRawMsg, &c)
	if err != nil {
		return err
	}

	db, dbOpenErr := NewLogisticsDb(c.DbFile)
	if dbOpenErr != nil {
		return dbOpenErr
	}
	l4g.Debug("Open logistics DB successful: %s", c.DbFile)
	self.logisticsdb = db
	self.config = &c
	self.pushMsgChannel = pushCh
	self.cron = cron.New()
	self.cron.AddFunc(c.LogisticsUpdateCron, func() {
		self.updateAndNotifyChangedLogistics()
	})
	self.commandMap = map[string]processFunc{
		"sublogi":        (*LogisticsService).subLogi,
		"unsublogi":      (*LogisticsService).unsubLogi,
		"getlogi":        (*LogisticsService).getLogi,
		"getrecentsub":   (*LogisticsService).getRecentSubs,
		"getcurrentlogi": (*LogisticsService).getCurrentLogi,
		"getcom":         (*LogisticsService).getCompany,
	}
	self.aliasCommandMap = map[string]string{
		"跟踪快递": "sublogi",
		"取消快递": "unsublogi",
		"查询快递": "getlogi",
		"快递记录": "getrecentsub",
		"我的快递": "getcurrentlogi",
		"物流公司": "getcom",
	}
	return nil
}

func (self *LogisticsService) StartService() error {
	self.cron.Start()
	self.started = true
	return nil
}

func (self *LogisticsService) IsStarted() bool {
	return self.started
}

func (self *LogisticsService) Stop() error {
	self.cron.Stop()
	err := self.logisticsdb.Close()
	self.started = false
	return err
}

func (self *LogisticsService) GetHelpMessage() string {
	var buffer bytes.Buffer
	for command, helpMsg := range commandHelp {
		buffer.WriteString(fmt.Sprintf("[%s]: %s\n", command, helpMsg))
	}
	return buffer.String()
}

func (self *LogisticsService) CommandFilter(command string, args []string) bool {
	if _, ok := self.aliasCommandMap[command]; ok {
		return true
	}

	if _, ok := self.commandMap[command]; ok {
		return true
	}
	return false
}

func (self *LogisticsService) Handle(username, command string, args []string) (string, error) {
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

func (self *LogisticsService) subLogi(username string, args []string) (string, error) {
	if len(args) != 3 {
		return "", errors.New("缺少参数！")
	}
	logisticsName := args[0]
	com := args[1]
	logisticsId := args[2]

	company, check := self.checkCompany(com)
	if !check {
		return "", errors.New("错误的公司名称或代码!")
	}
	if err := self.SubscribeLogistics(username, logisticsId, company, logisticsName); err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *LogisticsService) checkCompany(company string) (string, bool) {
	for key, value := range companyMap {
		if company == key || company == value {
			return value, true
		}
	}
	return "", false
}

func (self *LogisticsService) unsubLogi(username string, args []string) (string, error) {
	argsLen := len(args)
	switch argsLen {
	case 1:
		name := args[0]
		if err := self.UnsubscribeLogisticsByName(username, name); err != nil {
			return "", err
		}
	case 2:
		com := args[0]
		company, check := self.checkCompany(com)
		if !check {
			return "", errors.New("错误的公司名称或代码!")
		}
		logisticsId := args[1]
		if err := self.UnsubscribeLogistics(username, logisticsId, company); err != nil {
			return "", err
		}
	default:
		return "", errors.New("参数错误！")
	}
	return "OK", nil
}

func (self *LogisticsService) getLogi(username string, args []string) (string, error) {
	argsLen := len(args)
	switch argsLen {
	case 1:
		name := args[0]
		recordEntities, err := self.GetCurrentLogisticsByName(username, name)
		if err != nil {
			return "", err
		}
		return self.formatLogiOutput(recordEntities), nil
	case 2:
		com := args[0]
		company, check := self.checkCompany(com)
		if !check {
			return "", errors.New("错误的公司名称或代码!")
		}
		logisticsId := args[1]
		recordEntities, err := self.GetCurrentLogistics(logisticsId, company)
		if err != nil {
			return "", err
		}
		return self.formatLogiOutput(recordEntities), nil
	}
	return "", errors.New("参数错误！")
}

func (self *LogisticsService) getCurrentLogi(username string, args []string) (string, error) {
	changedLogisticsInfos, err := self.GetAllDeliveringLogistics(username)
	if err != nil {
		return "", err
	}
	if len(changedLogisticsInfos) == 0 {
		return "无记录", nil
	}
	var buffer bytes.Buffer
	for _, changedInfo := range changedLogisticsInfos {
		progress := self.formatLogiOutput(changedInfo.NewRecords)
		messageContent := fmt.Sprintf("\n[%s]的物流信息:%s", changedInfo.LogisticsName, progress)
		buffer.WriteString(messageContent)
	}
	return buffer.String(), nil
}

func (self *LogisticsService) getCompany(username string, args []string) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\n")
	for company, comCode := range companyMap {
		buffer.WriteString(fmt.Sprintf("公司名称[%s] ===> 公司代码[%s]\n", company, comCode))
	}
	return buffer.String(), nil
}

func (self *LogisticsService) formatLogiOutput(records []LogisticsRecordEntity) string {
	if len(records) == 0 {
		return "无记录"
	}
	var buffer bytes.Buffer
	buffer.WriteString("\n")
	for _, record := range records {
		message := record.Context
		fTime := time.Unix(record.Time, 0).Format("2006-01-02 15:04:05")
		buffer.WriteString(fmt.Sprintf("[%s] %s\n", fTime, message))
	}
	return buffer.String()
}

func (self *LogisticsService) getRecentSubs(username string, args []string) (string, error) {
	subscriptions, err := self.GetUserSubscription(username, 30)
	if err != nil {
		return "", err
	}
	return self.formatSubsOutput(subscriptions), nil
}

func (self *LogisticsService) formatSubsOutput(subscriptions []map[string]string) string {
	if len(subscriptions) == 0 {
		return "无记录"
	}
	var buffer bytes.Buffer
	buffer.WriteString("\n")
	for _, sub := range subscriptions {
		buffer.WriteString(fmt.Sprintf("%s %s %s\n", sub["logisticsName"], sub["company"], sub["logisticsId"]))
	}
	return buffer.String()
}

func (self *LogisticsService) SubscribeLogistics(username, logisticsId, company, logisticsName string) error {
	l, getError := self.logisticsdb.GetLogisticsInfoByIdCompany(logisticsId, company)
	if getError != nil {
		return getError
	}
	if l == nil {
		l = &LogisticsInfoEntity{
			Id:             0,
			LogisticsId:    logisticsId,
			Company:        company,
			State:          -1,
			Message:        "",
			LastUpdateTime: -1}
		saveErr := self.logisticsdb.SaveLogisticsInfo(l)
		if saveErr != nil {
			return saveErr
		}
	}
	refByName, getRefByNameError := self.logisticsdb.GetUserLogisticsRefByName(username, logisticsName)
	if getRefByNameError != nil {
		return getRefByNameError
	}
	if refByName != nil {
		errMsg := fmt.Sprintf("物流名称[%s]重复！", logisticsName)
		return errors.New(errMsg)
	}

	ref, getRefError := self.logisticsdb.GetUserLogisticsRef(username, l.Id)
	if getRefError != nil {
		return getRefError
	}
	if ref == nil {
		ref = &UserLogisticsRef{
			Id:                    0,
			Username:              username,
			LogisticsInfoEntityId: l.Id,
			LogisticsName:         logisticsName,
			Subscribe:             1}
	}
	ref.LogisticsName = logisticsName
	ref.Subscribe = 1
	saveErr := self.logisticsdb.SaveUserLogisticsRef(ref)
	if saveErr != nil {
		return saveErr
	}
	return nil
}

func (self *LogisticsService) GetUserSubscription(username string, limit int) ([]map[string]string, error) {
	refs, getError := self.logisticsdb.GetAllUserLogisticsRefs(username, limit)
	if getError != nil {
		return make([]map[string]string, 0), getError
	}
	if len(refs) == 0 {
		return make([]map[string]string, 0), nil
	}
	results := make([]map[string]string, 0)
	for _, ref := range refs {
		logisticsEntity, getEntityErr := self.logisticsdb.GetLogisticsInfoByEntityId(ref.LogisticsInfoEntityId)
		if getEntityErr != nil {
			return results, getEntityErr
		}
		logisticsInfoMap := make(map[string]string)
		logisticsInfoMap["logisticsName"] = ref.LogisticsName
		logisticsInfoMap["company"] = logisticsEntity.Company
		logisticsInfoMap["logisticsId"] = logisticsEntity.LogisticsId
		results = append(results, logisticsInfoMap)
	}
	return results, nil
}

func (self *LogisticsService) UnsubscribeLogistics(username, logisticsId, company string) error {
	ref, getRefError := self.logisticsdb.GetUserLogisticsRefByIdCompany(username, logisticsId, company)
	if getRefError != nil {
		return getRefError
	}
	if ref == nil {
		return errors.New("未订阅该快递！")
	}
	ref.Subscribe = 2
	saveErr := self.logisticsdb.SaveUserLogisticsRef(ref)
	if saveErr != nil {
		return saveErr
	}
	return nil
}

func (self *LogisticsService) UnsubscribeLogisticsByName(username, logisticsName string) error {
	ref, getRefError := self.logisticsdb.GetUserLogisticsRefByName(username, logisticsName)
	if getRefError != nil {
		return getRefError
	}
	if ref == nil {
		return errors.New("未订阅该快递！")
	}
	ref.Subscribe = 2
	saveErr := self.logisticsdb.SaveUserLogisticsRef(ref)
	if saveErr != nil {
		return saveErr
	}
	return nil
}

func (self *LogisticsService) updateAndNotifyChangedLogistics() {
	limit := 100
	for {
		entities, err := self.logisticsdb.GetUnfinishedLogistic(self.config.BeforeLastUpdate, limit)
		if err != nil {
			l4g.Error("GetUnfinishedLogistic error: %v", err)
			return
		}

		for _, entity := range entities {
			newRecords, updateErr := self.updateLogisticsProgress(&entity)
			if updateErr != nil {
				l4g.Error("UpdateLogisticsProgress error: %v", updateErr)
				continue
			}
			recLen := len(newRecords)
			if recLen == 0 {
				l4g.Debug("no new records in logistics id:%s, com: %s",
					entity.LogisticsId, entity.Company)
				continue
			}
			sort.Sort(byTime(newRecords))
			userRefs, getRefsErr := self.logisticsdb.GetSubUserLogisticsRefs(entity.Id)
			if getRefsErr != nil {
				l4g.Error("GetUserLogisticsRefs error: %v", getRefsErr)
				continue
			}
			for _, ref := range userRefs {
				progress := self.formatLogiOutput(newRecords)
				messageContent := fmt.Sprintf("\n[%s]有新的物流信息:%s", ref.LogisticsName, progress)
				pushMsg := &service.PushMessage{}
				pushMsg.Type = service.Notification
				pushMsg.Username = ref.Username
				pushMsg.Message = messageContent
				self.pushMsgChannel <- pushMsg
			}

		}
		if len(entities) < limit { // no more logistics need to be updated
			return
		}
	}
}

// return new record
func (self *LogisticsService) updateLogisticsProgress(lEntity *LogisticsInfoEntity) ([]LogisticsRecordEntity, error) {
	logisticsInfo, queryErr := Query(lEntity.Company, lEntity.LogisticsId)
	if queryErr != nil {
		return []LogisticsRecordEntity{}, queryErr
	}

	if logisticsInfo.Status != "200" {
		if (time.Now().Unix() - lEntity.CrtDate) > LOGISTICS_UPDATE_TIMEOUT {
			// timeout, make state = 701:error, don't update again
			lEntity.State = 701
		}
		lEntity.Message = logisticsInfo.Message
		updateErr := self.logisticsdb.SaveLogisticsInfo(lEntity)
		if updateErr != nil {
			return []LogisticsRecordEntity{}, updateErr
		}
		errMsg := fmt.Sprintf("Query logistics [%s %s]error: %s", lEntity.Company, lEntity.LogisticsId, logisticsInfo.Message)
		return []LogisticsRecordEntity{}, errors.New(errMsg)
	}

	lastUpdateTime := lEntity.LastUpdateTime
	var latestRecTime int64 = lastUpdateTime
	var records []LogisticsRecordEntity
	for _, rec := range logisticsInfo.Data {
		recTime, parseErr := time.Parse("2006-01-02 15:04:05", rec.Time)
		if parseErr != nil {
			return []LogisticsRecordEntity{}, parseErr
		}
		localT := time.Date(recTime.Year(), recTime.Month(), recTime.Day(),
			recTime.Hour(), recTime.Minute(), recTime.Second(), recTime.Nanosecond(),
			time.Local)
		rT := localT.Unix()
		if rT > lastUpdateTime {
			recEntity := &LogisticsRecordEntity{
				LogisticsInfoEntityId: lEntity.Id,
				Context:               rec.Context,
				Time:                  rT}
			saveErr := self.logisticsdb.SaveLogisticsRecord(recEntity)
			if saveErr != nil {
				return []LogisticsRecordEntity{}, saveErr
			}
			records = append(records, *recEntity)
			if rT > latestRecTime {
				latestRecTime = rT
			}
		}
	}
	s, _ := strconv.Atoi(logisticsInfo.State)
	lEntity.State = s
	lEntity.LastUpdateTime = latestRecTime
	updateErr := self.logisticsdb.SaveLogisticsInfo(lEntity)
	if updateErr != nil {
		return []LogisticsRecordEntity{}, updateErr
	}
	return records, nil
}

func (self *LogisticsService) GetCurrentLogisticsByName(username, name string) ([]LogisticsRecordEntity, error) {
	refByName, getRefByNameError := self.logisticsdb.GetUserLogisticsRefByName(username, name)
	if getRefByNameError != nil {
		return []LogisticsRecordEntity{}, getRefByNameError
	}
	if refByName == nil {
		errMsg := fmt.Sprintf("物流单[%s]不存在！", name)
		return []LogisticsRecordEntity{}, errors.New(errMsg)
	}
	records, err := self.logisticsdb.GetLogisticsRecords(refByName.LogisticsInfoEntityId)
	if err != nil {
		return []LogisticsRecordEntity{}, err
	}
	sort.Sort(byTime(records))
	return records, nil
}

func (self *LogisticsService) GetCurrentLogistics(logisticsId, company string) ([]LogisticsRecordEntity, error) {
	lEntity, getError := self.logisticsdb.GetLogisticsInfoByIdCompany(logisticsId, company)
	if getError != nil {
		return []LogisticsRecordEntity{}, getError
	}
	if lEntity != nil {
		records, err := self.logisticsdb.GetLogisticsRecords(lEntity.Id)
		if err != nil {
			return []LogisticsRecordEntity{}, err
		}
		sort.Sort(byTime(records))
		return records, nil
	}

	logisticsInfo, queryErr := Query(company, logisticsId)
	if queryErr != nil {
		return []LogisticsRecordEntity{}, queryErr
	}
	if logisticsInfo.Status != "200" {
		return []LogisticsRecordEntity{}, errors.New("物流查询错误: " + logisticsInfo.Message)
	}
	var records []LogisticsRecordEntity
	for _, rec := range logisticsInfo.Data {
		recTime, _ := time.Parse("2006-01-02 15:04:05", rec.Time)
		localT := time.Date(recTime.Year(), recTime.Month(), recTime.Day(),
			recTime.Hour(), recTime.Minute(), recTime.Second(), recTime.Nanosecond(),
			time.Local)
		rT := localT.Unix()
		recEntity := LogisticsRecordEntity{
			Context: rec.Context,
			Time:    rT}
		records = append(records, recEntity)
	}
	sort.Sort(byTime(records))
	return records, nil
}

func (self *LogisticsService) GetAllDeliveringLogistics(username string) ([]logisticsTrackingInfo, error) {
	refs, getRefsError := self.logisticsdb.GetAllDeliveringLogistics(username)
	if getRefsError != nil {
		return nil, getRefsError
	}
	allTrackingInfo := make([]logisticsTrackingInfo, 0)
	for _, ref := range refs {
		records, err := self.logisticsdb.GetLogisticsRecords(ref.LogisticsInfoEntityId)
		if err != nil {
			return nil, err
		}
		sort.Sort(byTime(records))
		trackingInfo := logisticsTrackingInfo{username, ref.LogisticsName, records}
		allTrackingInfo = append(allTrackingInfo, trackingInfo)
	}
	return allTrackingInfo, nil
}
