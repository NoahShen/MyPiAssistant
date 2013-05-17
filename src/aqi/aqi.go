package aqi

import ()

type DataSource int

const (
	USEmbassy DataSource = 1 + iota
	CNOfficial
)

type AqiData struct {
	City       string
	Aqi        int
	Time       int64
	Datasource DataSource
}
