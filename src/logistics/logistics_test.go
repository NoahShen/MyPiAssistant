package logistics

import (
	"testing"
)

func NoTestSubscribeLogistics(t *testing.T) {
	service, err := NewLogisticsService("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	subErr := service.SubscribeLogistics("Noah@example.com", "668031148649", "shentong", "goods")
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
	records, getErr := service.GetCurrentLogistics("668031148649", "shentong")
	if getErr != nil {
		t.Fatal(getErr)
	}
	t.Log("records:", records)
}

func TestUpdateAndGetChangedLogistics(t *testing.T) {
	service, err := NewLogisticsService("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	logisticsCh := make(chan *ChangedLogisticInfo, 1)
	go service.UpdateAndGetChangedLogistics(logisticsCh)
	for changedInfo := range logisticsCh {
		t.Log("changedInfo:", changedInfo)
	}

}
