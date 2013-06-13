package aqi

import (
	"fmt"
	"testing"
)

func createDBHelper(t *testing.T) *AqiDbHelper {
	Debug = true
	aqiDbhelper, err := NewAqiDbHelper("../../db/piaqidata.db")
	if err != nil {
		t.Fatal(err)
	}
	return aqiDbhelper
}

func _TestGetLatestAqiEntity(t *testing.T) {
	aqiDbhelper := createDBHelper(t)
	aqiDataEntity, err := aqiDbhelper.GetLatestAqiEntity("shanghai")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(aqiDataEntity)
}

func _TestSaveAqiData(t *testing.T) {
	aqiDbhelper := createDBHelper(t)
	aqiSh, fetchErr := FetchAqiFromWeb("shanghai")
	if fetchErr != nil {
		t.Fatal(fetchErr)
	}
	entitySh := convertAqiDataToEntity(aqiSh[0])
	saveShError := aqiDbhelper.SaveAqiDataEntity(entitySh)
	if saveShError != nil {
		t.Fatal(saveShError)
	}
	aqiBj, fetchErr1 := FetchAqiFromWeb("beijing")
	if fetchErr1 != nil {
		t.Fatal(fetchErr1)
	}
	entityBj := convertAqiDataToEntity(aqiBj[0])
	saveBjError := aqiDbhelper.SaveAqiDataEntity(entityBj)
	if saveBjError != nil {
		t.Fatal(saveBjError)
	}

	aqiSz, fetchErr2 := FetchAqiFromWeb("suzhou")
	if fetchErr2 != nil {
		t.Fatal(fetchErr2)
	}
	entitySz := convertAqiDataToEntity(aqiSz[0])
	saveSzError := aqiDbhelper.SaveAqiDataEntity(entitySz)
	if saveSzError != nil {
		t.Fatal(saveSzError)
	}
}

func convertAqiDataToEntity(aqiData *AqiData) *AqiDataEntity {
	entity := &AqiDataEntity{}
	entity.City = aqiData.City
	entity.Aqi = aqiData.Aqi
	entity.Time = aqiData.Time
	entity.Datasource = int(aqiData.Datasource)
	return entity
}
