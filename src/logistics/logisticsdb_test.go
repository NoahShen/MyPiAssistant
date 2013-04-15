package logistics

import (
	"testing"
)

func NoTestAddLogisticsInfo(t *testing.T) {
	db, err := NewLogisticsDb("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	logisticsInfoEntity := &LogisticsInfoEntity{LogisticsId: "668031148649", Company: "shentong"}
	l, addErr := db.AddLogisticsInfo(logisticsInfoEntity)
	if addErr != nil {
		t.Fatal(addErr)
	}
	t.Log("logistic info:", l)
}

func TestUpdateLogisticsRecords(t *testing.T) {
	db, err := NewLogisticsDb("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	updateErr := db.UpdateLogisticsStatus(1)
	if updateErr != nil {
		t.Fatal(updateErr)
	}
}
