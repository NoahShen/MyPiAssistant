package logistics

import (
	l4g "code.google.com/p/log4go"
	"testing"
)

func NoTestSubscribeLogistics(t *testing.T) {
	service, err := NewLogisticsService("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	subErr := service.SubscribeLogistics("Noah@example.com", "668031148649", "shentong", "test1")
	if subErr != nil {
		t.Fatal(subErr)
	}
	t.Log("Subscribe successful!")
}

//func NoTestUpdateLogisticsRecords(t *testing.T) {
//	service, err := NewLogisticsService("../../db/pilogistics.db")
//	if err != nil {
//		t.Fatal(err)
//	}
//	newRecords, updateErr := service.updateLogisticsProgress(1)
//	if updateErr != nil {
//		t.Fatal(updateErr)
//	}
//	t.Log("newRecords:", newRecords)
//}

func NoTestUnsubscribeLogistics(t *testing.T) {
	service, err := NewLogisticsService("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	subErr := service.UnsubscribeLogistics("Noah2@example.com", "668031148649", "shentong")
	if subErr != nil {
		t.Fatal(subErr)
	}
	t.Log("Unsubscribe successful!")
}

func NoTestGetCurrentLogistics(t *testing.T) {
	service, err := NewLogisticsService("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	//records, getErr := service.GetCurrentLogistics("1200722815552", "yunda")
	//records, getErr := service.GetCurrentLogistics("668031148649", "shentong")
	records, getErr := service.GetCurrentLogistics("966053314784", "shunfeng")
	if getErr != nil {
		t.Fatal(getErr)
	}
	for _, rec := range records {
		t.Log(rec)
	}
}

func NoTestUpdateAndGetChangedLogistics(t *testing.T) {
	l4g.LoadConfiguration("../config/logConfig.xml")
	service, err := NewLogisticsService("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	logisticsCh := make(chan *ChangedLogisticsInfo, 1)
	go service.UpdateAndGetChangedLogistics(logisticsCh)
	for changedInfo := range logisticsCh {
		t.Log("username:", changedInfo.Username)
		for _, rec := range changedInfo.NewRecords {
			t.Log(rec)
		}
	}

}
