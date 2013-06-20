package foodprice

import ()

var Debug = false

type CityFoodPrice struct {
	City     string
	Time     int64
	Food     string
	Unit     string
	AvgPrice float64
	MaxPrice float64
	MaxSite  string
	MinPrice float64
	MinSite  string
}

type DistrictFoodPrice struct {
	District    string
	Time        int64
	Food        string
	PricesSites []PriceSite
}

type PriceSite struct {
	Price float64
	Site  string
}
