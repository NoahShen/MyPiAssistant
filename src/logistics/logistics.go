package logistics

import (
	"bytes"
	l4g "code.google.com/p/log4go"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	LOGISTICS_UPDATE_TIMEOUT = 1 * 24 * 60 * 60
)

var company = map[string]string{
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

type ChangedLogisticsInfo struct {
	Username      string
	LogisticsName string
	NewRecords    []LogisticsRecordEntity
}

type LogisticsService struct {
	logisticsdb      *LogisticsDb
	commandMap       map[string]processFunc
	commandHelp      map[string]string
	voiceCommandMap  map[string]string
	beforeLastUpdate int64
}

func NewLogisticsService(dbFile string, beforeLastUpdate int64) (*LogisticsService, error) {
	db, dbOpenErr := NewLogisticsDb(dbFile)
	if dbOpenErr != nil {
		return nil, dbOpenErr
	}
	l4g.Debug("Open logistics DB successful: %s", dbFile)
	service := &LogisticsService{}
	service.logisticsdb = db
	service.beforeLastUpdate = beforeLastUpdate
	service.commandMap = map[string]processFunc{
		"sublogi":      (*LogisticsService).sublogi,
		"unsublogi":    (*LogisticsService).unsublogi,
		"getlogi":      (*LogisticsService).getlogi,
		"getrecentsub": (*LogisticsService).getRecentSubs,
		"getalllogi":   (*LogisticsService).getAlllogi,
		"getcom":       (*LogisticsService).getCompany,
	}
	service.voiceCommandMap = map[string]string{
		"物流查询": "getalllogi",
		"物流公司": "getcom",
	}
	service.commandHelp = map[string]string{
		"sublogi":      "subscribe one logistics, like sublogi name company logistics id",
		"unsublogi":    "unsubscribe one logistics, like unsublogi name or unsublogi company logistics id",
		"getlogi":      "get current logistics message, like getlogi name or getlogi company logistics id",
		"getrecentsub": "get recent subscribed logistics info",
		"getalllogi":   "get all delivering logistics info",
		"getCom":       "get all supported company",
	}
	return service, nil
}

func (self *LogisticsService) Close() error {
	return self.logisticsdb.Close()
}

func (self *LogisticsService) GetServiceName() string {
	return "Logistics Query"
}
func (self *LogisticsService) VoiceToCommand(voiceText string) string {
	comm := self.voiceCommandMap[voiceText]
	return comm
}

func (self *LogisticsService) GetComandHelp() string {
	var buffer bytes.Buffer
	for command, helpMsg := range self.commandHelp {
		buffer.WriteString(fmt.Sprintf("[%s]: %s\n", command, helpMsg))
	}
	buffer.WriteString("voice command:\n")
	for voice, command := range self.voiceCommandMap {
		buffer.WriteString(fmt.Sprintf("[%s] ===> %s\n", voice, command))
	}
	return buffer.String()
}

func (self *LogisticsService) CheckCommandType(command string) bool {
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	c := strings.ToLower(comm)
	for commandKey, _ := range self.commandMap {
		if strings.HasPrefix(c, commandKey) {
			l4g.Debug("[%s] is logistic command", command)
			return true
		}
	}
	return false
}

func (self *LogisticsService) Process(username, command string) (string, error) {
	commArr := strings.Split(command, " ")
	comm := commArr[0]
	f := self.commandMap[comm]
	if f == nil {
		return "", errors.New("Invalided logistic command!")
	}
	return f(self, username, commArr[1:])
}

func (self *LogisticsService) sublogi(username string, args []string) (string, error) {
	if len(args) != 3 {
		return "", errors.New("Missing args, command should be like: sublogi logisticsName company logisticsId")
	}
	logisticsName := args[0]
	company := args[1]
	logisticsId := args[2]

	if err := self.SubscribeLogistics(username, logisticsId, company, logisticsName); err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *LogisticsService) unsublogi(username string, args []string) (string, error) {
	argsLen := len(args)
	switch argsLen {
	case 1:
		name := args[0]
		if err := self.UnsubscribeLogisticsByName(username, name); err != nil {
			return "", err
		}
	case 2:
		company := args[0]
		logisticsId := args[1]
		if err := self.UnsubscribeLogistics(username, logisticsId, company); err != nil {
			return "", err
		}
	default:
		return "", errors.New("Please input company, logisticsId or input logisticsName.")
	}
	return "OK", nil
}

func (self *LogisticsService) getlogi(username string, args []string) (string, error) {
	argsLen := len(args)
	switch argsLen {
	case 1:
		name := args[0]
		recordEntities, err := self.GetCurrentLogisticsByName(username, name)
		if err != nil {
			return "", err
		}
		return self.FormatLogiOutput(recordEntities), nil
	case 2:
		company := args[0]
		logisticsId := args[1]
		recordEntities, err := self.GetCurrentLogistics(logisticsId, company)
		if err != nil {
			return "", err
		}
		return self.FormatLogiOutput(recordEntities), nil
	}
	return "", errors.New("Please input company, logisticsId or input logisticsName.")
}

func (self *LogisticsService) getAlllogi(username string, args []string) (string, error) {
	changedLogisticsInfos, err := self.GetAllDeliveringLogistics(username)
	if err != nil {
		return "", err
	}
	if len(changedLogisticsInfos) == 0 {
		return "no records", nil
	}
	var buffer bytes.Buffer
	for _, changedInfo := range changedLogisticsInfos {
		progress := self.FormatLogiOutput(changedInfo.NewRecords)
		messageContent := fmt.Sprintf("\nThe logistics of [%s]:%s", changedInfo.LogisticsName, progress)
		buffer.WriteString(messageContent)
	}
	return buffer.String(), nil
}

func (self *LogisticsService) getCompany(username string, args []string) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\n")
	for company, comCode := range company {
		buffer.WriteString(fmt.Sprintf("[%s] ===> %s\n", company, comCode))
	}
	return buffer.String(), nil
}

func (self *LogisticsService) FormatLogiOutput(records []LogisticsRecordEntity) string {
	if len(records) == 0 {
		return "no records"
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
		return "no records"
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
		errMsg := fmt.Sprintf("LogisticsName[%s] is duplicated!", logisticsName)
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
		return errors.New("The subscription not exist!")
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
		return errors.New("The subscription not exist!")
	}
	ref.Subscribe = 2
	saveErr := self.logisticsdb.SaveUserLogisticsRef(ref)
	if saveErr != nil {
		return saveErr
	}
	return nil
}

func (self *LogisticsService) UpdateAndGetChangedLogistics(logisticsCh chan<- *ChangedLogisticsInfo) {
	limit := 100
	defer close(logisticsCh)
	for {
		entities, err := self.logisticsdb.GetUnfinishedLogistic(self.beforeLastUpdate, limit)
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
				changedInfo := &ChangedLogisticsInfo{ref.Username, ref.LogisticsName, newRecords}
				logisticsCh <- changedInfo
			}

		}
		l := len(entities)
		if l < limit { // no more logistics need to be updated
			l4g.Debug("no more logistics: %d", l)
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
	var latestRecTime int64 = -1
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
		errMsg := fmt.Sprintf("LogisticsName[%s] not exist!", name)
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
		return []LogisticsRecordEntity{}, errors.New("Query logistics error: " + logisticsInfo.Message)
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

func (self *LogisticsService) GetAllDeliveringLogistics(username string) ([]ChangedLogisticsInfo, error) {
	refs, getRefsError := self.logisticsdb.GetAllDeliveringLogistics(username)
	if getRefsError != nil {
		return []ChangedLogisticsInfo{}, getRefsError
	}
	allChangedLogisticsInfo := make([]ChangedLogisticsInfo, 0)
	for _, ref := range refs {
		records, err := self.logisticsdb.GetLogisticsRecords(ref.LogisticsInfoEntityId)
		if err != nil {
			return []ChangedLogisticsInfo{}, err
		}
		sort.Sort(byTime(records))
		changedLogisticsInfo := ChangedLogisticsInfo{username, ref.LogisticsName, records}
		allChangedLogisticsInfo = append(allChangedLogisticsInfo, changedLogisticsInfo)
	}
	return allChangedLogisticsInfo, nil
}
