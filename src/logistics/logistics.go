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

type ChangedLogisticInfo struct {
	Username   string
	NewRecords []LogisticsRecordEntity
}

type LogisticsService struct {
	logisticsdb *LogisticsDb
	commandMap  map[string]processFunc
}

func NewLogisticsService(dbFile string) (*LogisticsService, error) {
	db, dbOpenErr := NewLogisticsDb(dbFile)
	if dbOpenErr != nil {
		return nil, dbOpenErr
	}
	l4g.Debug("Open logistics DB successful: %s", dbFile)
	service := &LogisticsService{}
	service.logisticsdb = db

	service.commandMap = map[string]processFunc{
		"sublogi":   (*LogisticsService).sublogi,
		"unsublogi": (*LogisticsService).unsublogi,
		"getlogi":   (*LogisticsService).getlogi,
	}
	return service, nil
}

func (self *LogisticsService) Close() error {
	return self.logisticsdb.Close()
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
	company := args[0]
	logisticsId := args[1]
	logisticsName := args[2]
	if err := self.SubscribeLogistics(username, logisticsId, company, logisticsName); err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *LogisticsService) unsublogi(username string, args []string) (string, error) {
	company := args[0]
	logisticsId := args[1]
	if err := self.UnsubscribeLogistics(username, logisticsId, company); err != nil {
		return "", err
	}
	return "OK", nil
}

func (self *LogisticsService) getlogi(username string, args []string) (string, error) {
	company := args[0]
	logisticsId := args[1]
	recordEntities, err := self.GetCurrentLogistics(logisticsId, company)
	if err != nil {
		return "", err
	}
	return self.formatLogiOutput(recordEntities), nil
}

func (self *LogisticsService) formatLogiOutput(records []LogisticsRecordEntity) string {
	if len(records) == 0 {
		return "no records"
	}
	var buffer bytes.Buffer
	buffer.WriteString("\n")
	for _, record := range records {
		message := record.Context
		fTime := time.Unix(record.Time, 0).Format("2006-01-02 15:04:05")
		buffer.WriteString(fmt.Sprintf("%s  %s\n", fTime, message))
	}
	return buffer.String()
}

func (self *LogisticsService) SubscribeLogistics(username, logisticsId, company, logisticsName string) error {
	l, getError := self.logisticsdb.GetLogisticsInfoByIdCompany(logisticsId, company)
	if getError != nil {
		return getError
	}
	if l == nil {
		l = &LogisticsInfoEntity{0, logisticsId, company, -1, "", -1}
		saveErr := self.logisticsdb.SaveLogisticsInfo(l)
		if saveErr != nil {
			return saveErr
		}
	}
	ref, getRefError := self.logisticsdb.GetUserLogisticsRef(username, l.Id)
	if getRefError != nil {
		return getRefError
	}
	if ref == nil {
		ref = &UserLogisticsRef{0, username, l.Id, logisticsName, 1}
	}
	ref.LogisticsName = logisticsName
	ref.Subscribe = 1
	saveErr := self.logisticsdb.SaveUserLogisticsRef(ref)
	if saveErr != nil {
		return saveErr
	}
	return nil
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

func (self *LogisticsService) UpdateAndGetChangedLogistics(logisticsCh chan<- *ChangedLogisticInfo) {
	startTime := time.Now().Unix()
	limit := 100
	defer close(logisticsCh)
	for {
		entities, err := self.logisticsdb.GetUnfinishedLogistic(startTime, limit)
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
			userRefs, getRefsErr := self.logisticsdb.GetUserLogisticsRefs(entity.Id)
			if getRefsErr != nil {
				l4g.Error("GetUserLogisticsRefs error: %v", getRefsErr)
				continue
			}
			for _, ref := range userRefs {
				changedInfo := &ChangedLogisticInfo{ref.Username, newRecords}
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
		// TODO don't update invalid logistics id again, make state to 701:error
		errMsg := fmt.Sprintf("Query logistics [%s %s]error: %s", lEntity.Company, lEntity.LogisticsId, logisticsInfo.Message)
		return []LogisticsRecordEntity{}, errors.New(errMsg)
	}

	lastUpdateTime := lEntity.LastUpdateTime

	s, _ := strconv.Atoi(logisticsInfo.State)
	lEntity.State = s
	lEntity.LastUpdateTime = time.Now().Unix()
	updateErr := self.logisticsdb.SaveLogisticsInfo(lEntity)
	if updateErr != nil {
		return []LogisticsRecordEntity{}, updateErr
	}

	var records []LogisticsRecordEntity
	for _, rec := range logisticsInfo.Data {
		recTime, parseErr := time.Parse("2006-01-02 15:04:05", rec.Time)
		if parseErr != nil {
			return []LogisticsRecordEntity{}, parseErr
		}
		rT := recTime.Unix()
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
		}
	}

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
		recTime.In(time.Now().Location())
		rT := recTime.Unix()

		l4g.Debug("origin Time: %s, timeobj: %v, parse: %d, ", rec.Time, recTime, rT)
		recEntity := LogisticsRecordEntity{
			Context: rec.Context,
			Time:    rT}
		records = append(records, recEntity)
	}
	sort.Sort(byTime(records))
	return records, nil
}
