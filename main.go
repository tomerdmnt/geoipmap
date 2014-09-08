package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/nranchev/go-libGeoIP"
)

func main() {
	ipRe := regexp.MustCompile("((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)")
	scanner := bufio.NewScanner(os.Stdin)

	geoip, err := libgeo.Load("./GeoLiteCity.dat")
	if err != nil {
		log.Fatal(err)
	}

	for scanner.Scan() {
		if ip := ipRe.FindString(scanner.Text()); ip != "" {
			if location := geoip.GetLocationByIP(ip); location != nil {
				fmt.Println(location.CountryName, location.City, location.Latitude, location.Longitude)
			} else {
				fmt.Println("N\\A")
			}
		}
	}
}
