package foodprice

import (
	//"fmt"
	"testing"
)

func createDBHelper(t *testing.T) *FoodPriceDbHelper {
	Debug = true
	foodPriceDbHelper, err := NewFoodPriceDbHelper("../../db/pifoodprice.db")
	if err != nil {
		t.Fatal(err)
	}
	return foodPriceDbHelper
}

func _TestAddCityFoodPrice(t *testing.T) {
	prices, err := FetchCityFoodPrice("shanghai")
	if err != nil {
		t.Fatal(err)
	}
	dbHelper := createDBHelper(t)
	for _, price := range prices {
		dbHelper.AddCityFoodPrice(convertCityFoodPriceToEntity(price))
	}
}

func convertCityFoodPriceToEntity(foodPrice *CityFoodPrice) *CityFoodPriceEntity {
	entity := &CityFoodPriceEntity{}
	entity.City = foodPrice.City
	entity.Time = foodPrice.Time
	entity.Food = foodPrice.Food
	entity.Unit = foodPrice.Unit
	entity.AvgPrice = foodPrice.AvgPrice
	entity.MaxPrice = foodPrice.MaxPrice
	entity.MaxSite = foodPrice.MaxSite
	entity.MinPrice = foodPrice.MinPrice
	entity.MinSite = foodPrice.MinSite
	return entity
}

func TestAddDistrictFoodPrice(t *testing.T) {
	prices, err := FetchDistrictFoodPrice("qingpu")
	if err != nil {
		t.Fatal(err)
	}
	dbHelper := createDBHelper(t)
	for _, price := range prices {
		entities := convertDistrictFoodPriceToEntity(price)
		for _, e := range entities {
			dbHelper.AddDistrictFoodPrice(e)
		}

	}
}

func convertDistrictFoodPriceToEntity(foodPrice *DistrictFoodPrice) []*DistrictFoodPriceEntity {
	entities := make([]*DistrictFoodPriceEntity, 0)
	for _, p := range foodPrice.PricesSites {
		entity := &DistrictFoodPriceEntity{}
		entity.District = foodPrice.District
		entity.Time = foodPrice.Time
		entity.Food = foodPrice.Food
		entity.Price = p.Price
		entity.Site = p.Site
		entities = append(entities, entity)
	}
	return entities
}
