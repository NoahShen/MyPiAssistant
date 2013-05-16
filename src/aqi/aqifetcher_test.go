package aqi

import (
	"fmt"
	"testing"
	"time"
)

func TestFetch(t *testing.T) {
	aqi, err := FetchAqiFromWeb("shanghai")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("aqi: %d, time: %v, source: %d", aqi.Aqi, time.Unix(aqi.Time, 0), aqi.Datasource)
	fmt.Println()
}

func TestFetchFromCN(t *testing.T) {
	aqi, err := FetchAqiFromWeb("suzhou")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("aqi: %d, time: %v, source: %d", aqi.Aqi, time.Unix(aqi.Time, 0), aqi.Datasource)
	fmt.Println()
}
