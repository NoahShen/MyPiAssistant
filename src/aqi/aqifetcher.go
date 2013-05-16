package aqi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"
)

var usEmbassyMap = map[string]string{
	"beijing":   "10",
	"shanghai":  "21",
	"guangzhou": "20",
	"chengdu":   "28",
}

const (
	usEmbassyUrl  = "http://pm25.sinaapp.com/if/getaqis.php?city=%s&type=1"
	cnOfficialUrl = "http://pm25.in/api/querys/only_aqi.json?city=%s&token=FydAKx5y1BBbqXeLcxyi&stations=no"
)

type DataSource int

const (
	USEmbassy DataSource = 1 + iota
	CNOfficial
)

type AqiData struct {
	Aqi        int
	Time       int64
	Datasource DataSource
}

func FetchAqiFromWeb(city string) (*AqiData, error) {
	cityCode, ok := usEmbassyMap[city]
	if ok {
		return fetchAqiFromUSEmbassy(cityCode)
	}
	return fetchAqiFromCNOfficial(city)
}

type AqiCNOfficialItem struct {
	Aqi       int       `json:"aqi,omitempty"`
	Area      string    `json:"area,omitempty"`
	Quality   string    `json:"quality,omitempty"`
	TimePoint time.Time `json:"time_point,omitempty"`
}

func fetchAqiFromCNOfficial(city string) (*AqiData, error) {
	url := fmt.Sprintf(cnOfficialUrl, city)
	bytes, err := getHttpResponseContent(url)
	if err != nil {
		return nil, err
	}
	aqiItems := make([]AqiCNOfficialItem, 1)
	if unmarshalErr := json.Unmarshal(bytes, &aqiItems); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	aqiItem := aqiItems[0]
	aqiData := &AqiData{}
	aqiData.Aqi = aqiItem.Aqi
	//get timepoint is "2013-05-16T16:00:00Z", lack of timezone
	originalTime := aqiItem.TimePoint
	correntTime := time.Date(originalTime.Year(), originalTime.Month(), originalTime.Day(),
		originalTime.Hour(), originalTime.Minute(), originalTime.Second(), originalTime.Nanosecond(),
		time.Local)
	aqiData.Time = correntTime.Unix()
	aqiData.Datasource = CNOfficial
	return aqiData, nil
}

type AqiUSEmbassyItem struct {
	Aqi       int   `json:"i,omitempty"`
	TimeShort int64 `json:"t,omitempty"`
}

type byTime []AqiUSEmbassyItem

func (s byTime) Len() int {
	return len(s)
}

func (s byTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byTime) Less(i, j int) bool {
	return s[i].TimeShort < s[j].TimeShort
}

func fetchAqiFromUSEmbassy(cityCode string) (*AqiData, error) {
	url := fmt.Sprintf(usEmbassyUrl, cityCode)
	bytes, err := getHttpResponseContent(url)
	if err != nil {
		return nil, err
	}
	aqiItems := make([]AqiUSEmbassyItem, 1)
	if unmarshalErr := json.Unmarshal(bytes, &aqiItems); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	sort.Sort(byTime(aqiItems))
	aqiItem := aqiItems[len(aqiItems)-1]
	aqiData := &AqiData{}
	aqiData.Aqi = aqiItem.Aqi
	aqiData.Time = aqiItem.TimeShort * 3600
	aqiData.Datasource = USEmbassy
	return aqiData, nil
}

func getHttpResponseContent(url string) ([]byte, error) {
	queryReq, _ := http.NewRequest("GET", url, nil)
	queryReq.Header.Set("Accept", "*/*")
	queryReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.31 (KHTML, like Gecko) Chrome/26.0.1410.64 Safari/537.31")
	queryReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	queryResp, queryErr := http.DefaultClient.Do(queryReq)
	if queryErr != nil {
		return nil, queryErr
	}
	defer queryResp.Body.Close()
	return ioutil.ReadAll(queryResp.Body)
}
