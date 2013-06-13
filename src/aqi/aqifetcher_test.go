package aqi

import (
	"fmt"
	"testing"
	"time"
)

func _TestFetch(t *testing.T) {
	Debug = true
	aqi, err := FetchAqiFromWeb("shanghai")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range aqi {
		fmt.Printf("aqi: %d, time: %v, source: %d", a.Aqi, time.Unix(a.Time, 0), a.Datasource)
		fmt.Println()
	}
}

func _TestFetchFromCN(t *testing.T) {
	Debug = true
	aqi, err := FetchAqiFromWeb("suzhou")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range aqi {
		fmt.Printf("aqi: %d, time: %v, source: %d", a.Aqi, time.Unix(a.Time, 0), a.Datasource)
		fmt.Println()
	}
}
