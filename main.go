package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"

	"github.com/GeertJohan/go.rice"
	"github.com/skratchdot/open-golang/open"
	"github.com/tomerdmnt/go-libGeoIP"
)

type GIJSON struct {
	Data    map[string]*Country `json:"data"`
	Bubbles []*City             `json:"bubbles"`
	Total   int                 `json:"total"`
}

type Country struct {
	Name    string `json:"country"`
	FillKey string `json:"fillKey"`
}

type City struct {
	Country   string  `json:"country"`
	Name      string  `json:"city"`
	Radius    float64 `json:"radius"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	FillKey   string  `json:"fillKey"`
	Count     int     `json:"count"`
}

type ByRadius []*City

func (a ByRadius) Len() int           { return len(a) }
func (a ByRadius) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRadius) Less(i, j int) bool { return a[i].Radius < a[j].Radius }

var gijson *GIJSON = &GIJSON{Data: make(map[string]*Country), Bubbles: []*City{}}

func readStdin() {
	ipRe := regexp.MustCompile("((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)")
	scanner := bufio.NewScanner(os.Stdin)

	DBBox := rice.MustFindBox("db")
	dbFile, err := DBBox.Open("GeoLiteCity.dat")
	if err != nil {
		log.Fatal(err)
	}
	geoip, err := libgeo.LoadFromReader(dbFile)

	if err != nil {
		log.Fatal(err)
	}

	for scanner.Scan() {
		if ip := ipRe.FindString(scanner.Text()); ip != "" {
			if location := geoip.GetLocationByIP(ip); location != nil {
				processLocation(location)
			}
		}
	}
}

func processLocation(location *libgeo.Location) {
	var found bool = false

	gijson.Data[CountryCodes[location.CountryCode]] = &Country{Name: location.CountryName, FillKey: "accessed"}

	for _, b := range gijson.Bubbles {
		if b.Country == location.CountryName && b.Name == location.City {
			b.Count++
			found = true
			break
		}
	}
	if !found {
		city := &City{
			Country:   location.CountryName,
			Name:      location.City,
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
			Count:     1,
		}
		gijson.Bubbles = append(gijson.Bubbles, city)
	}
	gijson.Total++
}

func handleGIData(w http.ResponseWriter, r *http.Request) {
	for _, b := range gijson.Bubbles {
		b.Radius = math.Sqrt(float64(b.Count * 5000 / gijson.Total))
		b.FillKey = fillKey(b.Count, gijson.Total)
	}
	sort.Sort(sort.Reverse(ByRadius(gijson.Bubbles)))
	json, err := json.Marshal(gijson)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func fillKey(count int, total int) string {
	percentage := count * 100 / total
	switch {
	case percentage >= 0 && percentage <= 2:
		return "0to2"
	case percentage > 2 && percentage <= 6:
		return "3to6"
	case percentage > 6 && percentage <= 10:
		return "7to10"
	case percentage > 10 && percentage <= 100:
		return "10to100"
	}
	return ""
}

func main() {
	http.HandleFunc("/gidata", handleGIData)
	http.Handle("/", http.FileServer(rice.MustFindBox("public").HTTPBox()))

	port, _ := strconv.Atoi(os.Getenv("PORT"))
	address := fmt.Sprintf("127.0.0.1:%d", port)

	go readStdin()

	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		addr := fmt.Sprintf("http://%s", l.Addr())
		fmt.Println(addr)
		open.Start(addr)
	}()
	log.Fatal(http.Serve(l, nil))
}
