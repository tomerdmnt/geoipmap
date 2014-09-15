package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"text/template"

	"github.com/GeertJohan/go.rice"
	"github.com/skratchdot/open-golang/open"
	"github.com/tomerdmnt/go-libGeoIP"
)

type GIJSON struct {
	Countries map[string]*Country `json:"countries"`
	Cities    []*City             `json:"cities"`
	Total     int                 `json:"total"`
}

type Country struct {
	Name string `json:"country"`
	Code string `json:"code"`
}

type City struct {
	Country   string  `json:"country"`
	Name      string  `json:"city"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Count     int     `json:"count"`
}

var gijson *GIJSON = &GIJSON{Countries: make(map[string]*Country), Cities: []*City{}}

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

	gijson.Countries[location.CountryName] = &Country{Name: location.CountryName}

	for _, c := range gijson.Cities {
		if c.Country == location.CountryName && c.Name == location.City {
			c.Count++
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
		gijson.Cities = append(gijson.Cities, city)
	}
	gijson.Total++
}

func handleGIData(w http.ResponseWriter, r *http.Request) {
	json, err := json.Marshal(gijson)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func serveIndex(title string) func(http.ResponseWriter, *http.Request) {
	var html bytes.Buffer

	tmpl, err := rice.MustFindBox("templates").String("index.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(&html, map[string]string{"Title": title}); err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(html.Bytes())
	}
}

func main() {
	title := flag.String("title", "", "Optional Title")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: [geoipmap] [-title <title>] [-h]

    geoipmap reads logs from stdin
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	http.HandleFunc("/gidata", handleGIData)
	http.Handle("/resources/", http.FileServer(rice.MustFindBox("public").HTTPBox()))
	http.HandleFunc("/", serveIndex(*title))

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
