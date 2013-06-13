package aqi

import ()

var Debug = false

type DataSource int

const (
	USEmbassy DataSource = 1 + iota
	CNOfficial
)

var DatasourceMap = map[DataSource]string{
	USEmbassy:  "美国大使馆",
	CNOfficial: "中国官方",
}

type AqiData struct {
	City       string
	Aqi        int
	Time       int64
	Datasource DataSource
}

type byTime []*AqiData

func (s byTime) Len() int {
	return len(s)
}

func (s byTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byTime) Less(i, j int) bool {
	return s[i].Time > s[j].Time
}
