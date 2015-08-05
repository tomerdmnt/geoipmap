package geoip

import (
	"net"
	"testing"
)

func BenchmarkLookupIP(b *testing.B) {
	geo, err := Open("GeoLite2-City.mmdb")
	if err != nil {
		b.Fatal(err)
	}
	ip := net.ParseIP("17.0.0.1")
	if ip == nil {
		b.Fatal("bad ip")
	}
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		geo.LookupIP(ip)
	}
}

func BenchmarkLookupIPValue(b *testing.B) {
	geo, err := Open("GeoLite2-City.mmdb")
	if err != nil {
		b.Fatal(err)
	}
	ip := net.ParseIP("17.0.0.1")
	if ip == nil {
		b.Fatal("bad ip")
	}
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		geo.LookupIPValue(ip)
	}
}
