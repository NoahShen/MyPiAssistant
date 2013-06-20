package foodprice

import (
	"fmt"
	"testing"
)

func _TestFetchCityFoodPrice(t *testing.T) {
	prices, err := FetchCityFoodPrice("shanghai")
	if err != nil {
		t.Fatal(err)
	}
	for _, price := range prices {
		fmt.Printf("record: %+v\n", price)
	}
}

func _TestFetchDistrictFoodPrice(t *testing.T) {
	prices, err := FetchDistrictFoodPrice("qingpu")
	if err != nil {
		t.Fatal(err)
	}
	for _, price := range prices {
		fmt.Printf("record: %+v\n", price)
	}
}
