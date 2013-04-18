package logistics

import (
	"testing"
)

func NoTestSaveLogisticsInfo(t *testing.T) {
	db, err := NewLogisticsDb("../../db/pilogistics.db")
	if err != nil {
		t.Fatal(err)
	}
	logisticsInfoEntity := &LogisticsInfoEntity{0, "668031148649", "shentong", -1, "", -1}
	addErr := db.SaveLogisticsInfo(logisticsInfoEntity)
	if addErr != nil {
		t.Fatal(addErr)
	}
	t.Log("logistic info:", logisticsInfoEntity)
}
