// +build compare

package geoip

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/oschwald/maxminddb-golang"
	"net"
	"testing"
)

func BenchmarkLookupOschwaldMaxminddb(b *testing.B) {
	db, err := maxminddb.Open("GeoLite2-City.mmdb")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	b.ResetTimer()
	ip := net.ParseIP("17.0.0.1")
	for ii := 0; ii < b.N; ii++ {
		db.Lookup(ip)
	}
}

func BenchmarkLookupOschwaldGeoip2(b *testing.B) {
	db, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	b.ResetTimer()
	ip := net.ParseIP("17.0.0.1")
	for ii := 0; ii < b.N; ii++ {
		db.City(ip)
	}
}
