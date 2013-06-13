package aqi

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"utils"
)

var usEmbassyMap = map[string]string{
	"beijing":   "1",
	"chengdu":   "2",
	"guangzhou": "3",
	"shanghai":  "4",
	"shenyang":  "5",
}

func FetchAqiFromWeb(city string) ([]*AqiData, error) {
	cityCode, ok := usEmbassyMap[city]
	if ok {
		return fetchAqiFromUSEmbassy(city, cityCode)
	}
	return fetchAqiFromCNOfficial(city)
}

const (
	cnOfficialUrl = "http://pm25.in/api/querys/only_aqi.json?city=%s&token=FydAKx5y1BBbqXeLcxyi&stations=no"
)

type AqiCNOfficialItem struct {
	Aqi       int    `json:"aqi,omitempty"`
	Area      string `json:"area,omitempty"`
	Quality   string `json:"quality,omitempty"`
	TimePoint string `json:"time_point,omitempty"`
}

func fetchAqiFromCNOfficial(city string) ([]*AqiData, error) {
	url := fmt.Sprintf(cnOfficialUrl, city)
	bytes, err := getHttpResponseContent(url)
	if err != nil {
		return nil, err
	}
	if Debug {
		fmt.Printf("***Get %s AQI from CN: %s\n", city, strings.TrimSpace(string(bytes)))
	}

	aqiItems := make([]AqiCNOfficialItem, 1)
	if unmarshalErr := json.Unmarshal(bytes, &aqiItems); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	aqiDataArr := make([]*AqiData, 0)
	for _, aqiItem := range aqiItems {
		aqiData := &AqiData{}
		aqiData.City = city
		aqiData.Aqi = aqiItem.Aqi
		//2006-01-02T15:04:05Z
		time, parseErr := utils.ConvertToUnixTime("2006-01-02T15:04:05Z", aqiItem.TimePoint)
		if parseErr != nil {
			return aqiDataArr, parseErr
		}
		aqiData.Time = time
		aqiData.Datasource = CNOfficial
		aqiDataArr = append(aqiDataArr, aqiData)
	}
	sort.Sort(byTime(aqiDataArr))
	return aqiDataArr, nil
}

const (
	usEmbassyUrl = "http://www.stateair.net/web/rss/1/%s.xml"
)

type rss2_0Feed struct {
	XMLName xml.Name       `xml:"rss"`
	Channel *rss2_0Channel `xml:"channel"`
}

type rss2_0Channel struct {
	XMLName     xml.Name     `xml:"channel"`
	Title       string       `xml:"title"`
	Description string       `xml:"description"`
	Link        string       `xml:"link"`
	Langueage   string       `xml:"language"`
	Ttl         int          `xml:"ttl"`
	Items       []rss2_0Item `xml:"item"`
}

type rss2_0Item struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	Link        string   `xml:"link"`
	Param       string   `xml:"Param"`
	Conc        float64  `xml:"Conc"`
	AQI         int      `xml:"AQI"`
	Desc        string   `xml:"Desc"`
	Date        string   `xml:"ReadingDateTime"`
}

func fetchAqiFromUSEmbassy(city, cityCode string) ([]*AqiData, error) {
	url := fmt.Sprintf(usEmbassyUrl, cityCode)
	bytes, err := getHttpResponseContent(url)
	if err != nil {
		return nil, err
	}
	if Debug {
		fmt.Printf("***Get %s AQI from USEmbassy: %s\n", city, strings.TrimSpace(string(bytes)))
	}
	feed := rss2_0Feed{}
	if unmarshalErr := xml.Unmarshal(bytes, &feed); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	aqiDataArr := make([]*AqiData, 0)
	rssChannel := feed.Channel
	if rssChannel != nil {
		rssItems := rssChannel.Items
		for _, rssItem := range rssItems {
			aqiData := &AqiData{}
			aqiData.City = city
			aqiData.Aqi = rssItem.AQI
			time, parseErr := utils.ConvertToUnixTime("01/02/2006 3:04:05 PM", rssItem.Date)
			if parseErr != nil {
				return aqiDataArr, parseErr
			}
			aqiData.Time = time
			aqiData.Datasource = USEmbassy
			aqiDataArr = append(aqiDataArr, aqiData)
		}
	}
	sort.Sort(byTime(aqiDataArr))

	return aqiDataArr, nil
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
