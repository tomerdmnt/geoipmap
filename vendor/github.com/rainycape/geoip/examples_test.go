package geoip_test

import (
	"fmt"
	"geoip"
)

func ExampleOpen() {
	db, err := geoip.Open("GeoLite2-City.mmdb.gz")
	if err != nil {
		panic(err)
	}
	res, err := db.Lookup("17.0.0.1")
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Country.Name)
	fmt.Println(res.City.Name)
	// Output:
	// United States
	// Cupertino
}
