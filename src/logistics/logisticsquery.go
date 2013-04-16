package logistics

import (
	l4g "code.google.com/p/log4go"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"utils"
)

const (
	QUERY_URL = "http://www.kuaidi100.com/query?type=%s&postid=%s&id=1&valicode=&temp=%f"
)

type LogisticsRecord struct {
	Context string `json:"context,omitempty"`
	Ftime   string `json:"ftime,omitempty"`
	Time    string `json:"time,omitempty"`
}

type LogisticsInfo struct {
	Com         string            `json:"com,omitempty"` // company
	Condition   string            `json:"condition,omitempty"`
	Data        []LogisticsRecord `json:"data,omitempty"`
	Message     string            `json:"message,omitempty"`
	LogisticsId string            `json:"nu,omitempty"`     // logistics id
	State       string            `json:"state,omitempty"`  //logistics status
	Status      string            `json:"status,omitempty"` // query status
	Updatetime  string            `json:"updatetime,omitempty"`
}

func Query(com, logisticsId string) (*LogisticsInfo, error) {
	//request like a browser
	url := fmt.Sprintf(QUERY_URL, com, logisticsId, utils.RandomFloat32)
	queryReq, _ := http.NewRequest("GET", url, nil)
	queryReq.Header.Set("Accept", "*/*")
	queryReq.Header.Set("Referer", "http://www.kuaidi100.com/")
	queryReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:17.0) Gecko/20100101 Firefox/17.0")
	queryReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	queryResp, queryErr := http.DefaultClient.Do(queryReq)
	if queryErr != nil {
		return nil, queryErr
	}
	defer queryResp.Body.Close()
	bytes, readErr := ioutil.ReadAll(queryResp.Body)
	if readErr != nil {
		return nil, readErr
	}
	l4g.Debug("Query logistics response: %s", strings.TrimSpace(string(bytes)))
	logisticsInfo := &LogisticsInfo{}
	if unmarshalErr := json.Unmarshal(bytes, logisticsInfo); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return logisticsInfo, nil
}
