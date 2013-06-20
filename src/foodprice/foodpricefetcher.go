package foodprice

import (
	"errors"
	"fmt"
	"github.com/NoahShen/goquery"
	"strconv"
	"strings"
	"time"
	"utils"
)

var CityMap = map[string]bool{
	"shanghai": true,
}

const (
	cityFoodPriceUrl = "http://www.shjjcd.gov.cn/www/more/"
)

func GetCityFoodPriceUrl(city string) (string, error) {
	_, ok := CityMap[city]
	if !ok {
		return "", errors.New("Not support this city yet!")
	}
	return cityFoodPriceUrl, nil
}

func FetchCityFoodPrice(city string) ([]*CityFoodPrice, error) {
	cityFoodPrices := make([]*CityFoodPrice, 0)
	url, urlErr := GetCityFoodPriceUrl(city)
	if urlErr != nil {
		return cityFoodPrices, urlErr
	}
	// do not use server cache
	url = fmt.Sprintf(url+"?_=%d", time.Now().UTC().UnixNano())
	var doc *goquery.Document
	var e error
	if doc, e = goquery.NewDocument(url); e != nil {
		return cityFoodPrices, e
	}

	acquisitionTimeSele := doc.Find("#maincontent li").FilterFunction(func(i int, s *goquery.Selection) bool {
		content := s.Text()
		return strings.Contains(content, "采价时间")
	})
	time, convertErr := convertPageTimeToTime(acquisitionTimeSele.Text())
	if convertErr != nil {
		return cityFoodPrices, errors.New("parse time error!")
	}

	doc.Find("#WebSitePrice tr").FilterFunction(func(i int, s *goquery.Selection) bool {
		return !s.HasClass("title")
	}).Each(func(i int, s *goquery.Selection) {
		fields := s.Find("td")
		cityFoodPrice := &CityFoodPrice{}
		cityFoodPrice.City = city
		cityFoodPrice.Time = time
		cityFoodPrice.Food = strings.TrimSpace(fields.Eq(0).Text())
		cityFoodPrice.Unit = strings.TrimSpace(fields.Eq(2).Text())
		cityFoodPrice.AvgPrice, _ = strconv.ParseFloat(fields.Eq(3).Text(), 64)
		cityFoodPrice.MaxPrice, _ = strconv.ParseFloat(fields.Eq(4).Text(), 64)
		cityFoodPrice.MaxSite = strings.TrimSpace(fields.Eq(5).Text())
		cityFoodPrice.MinPrice, _ = strconv.ParseFloat(fields.Eq(6).Text(), 64)
		cityFoodPrice.MinSite = strings.TrimSpace(fields.Eq(7).Text())
		cityFoodPrices = append(cityFoodPrices, cityFoodPrice)
	})
	return cityFoodPrices, nil
}

var DistrictMap = map[string]string{
	"pudong":    "310115",
	"huangpu":   "310101",
	"xuhui":     "310104",
	"changning": "310105",
	"jingan":    "310106",
	"putuo":     "310107",
	"zhabei":    "310108",
	"hongkou":   "310109",
	"yangpu":    "310110",
	"baoshan":   "310113",
	"minhang":   "310112",
	"jiading":   "310114",
	"jinshan":   "310116",
	"songjiang": "310117",
	"qingpu":    "310118",
	"fengxian":  "310120",
	"chongming": "310130",
}

const (
	districtFoodPriceUrl = "http://www.shjjcd.gov.cn/www/%s"
)

func GetDistrictFoodPriceUrl(district string) (string, error) {
	districtCode, ok := DistrictMap[district]
	if !ok {
		return "", errors.New("Not support this district yet!")
	}
	return fmt.Sprintf(districtFoodPriceUrl, districtCode), nil
}

func FetchDistrictFoodPrice(district string) ([]*DistrictFoodPrice, error) {
	districtFoodPrices := make([]*DistrictFoodPrice, 0)
	url, urlErr := GetDistrictFoodPriceUrl(district)
	if urlErr != nil {
		return districtFoodPrices, urlErr
	}
	url = fmt.Sprintf(url+"?_=%d", time.Now().UTC().UnixNano())
	var doc *goquery.Document
	var e error
	if doc, e = goquery.NewDocument(url); e != nil {
		return districtFoodPrices, e
	}

	acquisitionTimeSele := doc.Find("#maincontent li").FilterFunction(func(i int, s *goquery.Selection) bool {
		content := s.Text()
		return strings.Contains(content, "采价时间")
	})
	time, convertErr := convertPageTimeToTime(acquisitionTimeSele.Text())
	if convertErr != nil {
		return districtFoodPrices, errors.New("parse time error!")
	}

	titles := make([]string, 0)
	doc.Find("#WebSitePrice tr.title td").Each(func(i int, s *goquery.Selection) {
		titles = append(titles, strings.TrimSpace(s.Text()))
	})

	doc.Find("#WebSitePrice tr").FilterFunction(func(i int, s *goquery.Selection) bool {
		return !s.HasClass("title")
	}).Each(func(i int, s *goquery.Selection) {
		districtFoodPrice := &DistrictFoodPrice{}
		districtFoodPrice.District = district
		districtFoodPrice.Time = time
		s.Find("td").Each(func(tdI int, tdSele *goquery.Selection) {
			if tdI == 0 {
				districtFoodPrice.Food = strings.TrimSpace(tdSele.Text())
				return
			}
			priceSite := &PriceSite{}
			priceSite.Price, _ = strconv.ParseFloat(tdSele.Text(), 64)
			priceSite.Site = titles[tdI]
			districtFoodPrice.PricesSites = append(districtFoodPrice.PricesSites, *priceSite)
		})
		districtFoodPrices = append(districtFoodPrices, districtFoodPrice)
	})
	return districtFoodPrices, nil
}

func convertPageTimeToTime(content string) (int64, error) {
	// &nbsp;&nbsp;采价时间：2013-06-19上午9：00之前
	prefix := "采价时间："
	suffix := "之前"
	start := strings.Index(content, prefix) + len(prefix)
	if start < 0 {
		return -1, errors.New("Not found prefix")
	}
	end := strings.Index(content, suffix)
	if end < 0 {
		return -1, errors.New("Not found suffix")
	}
	dateTime := strings.Split(content[start:end], "上午")
	timeStr := fmt.Sprintf("%s %sAM", dateTime[0], dateTime[1])
	timeStr = strings.Replace(timeStr, "：", ":", -1)

	return utils.ConvertToUnixTime("2006-01-02 3:04PM", timeStr)
}
