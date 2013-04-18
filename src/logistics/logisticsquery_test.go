package logistics

import (
	l4g "code.google.com/p/log4go"
	"testing"
)

func NoTestQueryLogisticsInfo(t *testing.T) {
	l4g.LoadConfiguration("../config/logConfig.xml")
	logisticsInfo, err := Query("shentong", "668031148649")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(logisticsInfo)
}
