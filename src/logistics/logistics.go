package logistics

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

type LogisticsService struct {
	logisticsdb *LogisticsDb
}

func NewLogisticsService(dbFile string) (*LogisticsService, error) {
	db, dbOpenErr := NewLogisticsDb(dbFile)
	if dbOpenErr != nil {
		return nil, dbOpenErr
	}
	service := &LogisticsService{}
	service.logisticsdb = db
	return service, nil
}

func (self *LogisticsService) Close() error {
	return self.logisticsdb.Close()
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

// return new record
func (self *LogisticsService) updateLogisticsProgress(logisticsEntityId int) ([]LogisticsRecordEntity, error) {
	lEntity, getError := self.logisticsdb.GetLogisticsInfoByEntityId(logisticsEntityId)
	if getError != nil {
		return []LogisticsRecordEntity{}, getError
	}
	if lEntity == nil {
		return []LogisticsRecordEntity{}, errors.New(fmt.Sprintf("LogisticsInfo[%d] not exist!", logisticsEntityId))
	}
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
			// new record TODO may need to return
			recEntity := &LogisticsRecordEntity{
				LogisticsInfoEntityId: logisticsEntityId,
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
		rT := recTime.Unix()
		recEntity := LogisticsRecordEntity{
			Context: rec.Context,
			Time:    rT}
		records = append(records, recEntity)
	}
	return records, nil
}
