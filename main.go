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
	"github.com/tomerdmnt/geoipmap/vendor/github.com/rainycape/geoip"
	"github.com/yuin/gopher-lua"
)

var usage = `Usage: [geoipmap] [-title <title>] [-script <script>] [-addr <address>] [-h]

    geoipmap reads logs from stdin and displays the geo ip data on a world map

    EXAMPLES

      NGINX

        $ ssh root@server.com "zcat -f /var/log/nginx/access.log.* & tail -n 0 -F /var/log/nginx/access.log" | geoipmap -title "nginx access"

      SSH

        $ ssh root@server.com "zcat -f /var/log/auth.log.* & tail -n 0 -F /var/log/auth.log" | geoipmap -title "ssh access"

      FAIL2BAN

        $ ssh root@server.com "zcat -f /var/log/fail2ban.log* & tail -n 0 -F /var/log/fail2ban.log" | grep Ban | geoipmap -title "fail2ban"

    LUA SCRIPT EXAMPLE

        function record(r)
            if r.Country == "Greenland" then
                -- filter out ips from greenland
                return false
            end

            -- original log line
            print(r.Line)
            -- print values
            print(r.CountryCode)
            print(r.Country)
            print(r.City)    
            print(r.CityCode)    
            print(r.PostalCode)
            print(r.Latitude)
            print(r.Longitude)
            print(r.IP)

            -- the record can be altered
            r.Longitude = r.Longitude + 10
        end
`

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
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Count     int     `json:"count"`
}

type Record struct {
	CountryCode string
	Country     string
	Region      string
	City        string
	CityCode    string
	PostalCode  string
	Latitude    float64
	Longitude   float64
	IP          string
	Line        string
}

func newRecord(geoipRecord *geoip.Record, ip, line string) *Record {
	record := &Record{
		PostalCode: geoipRecord.PostalCode,
		Latitude:   geoipRecord.Latitude,
		Longitude:  geoipRecord.Longitude,
		IP:         ip,
		Line:       line,
	}
	if geoipRecord.Country != nil {
		record.Country = geoipRecord.Country.Name.String()
		record.CountryCode = geoipRecord.Country.Code
	}
	if geoipRecord.City != nil {
		record.City = geoipRecord.City.Name.String()
		record.CityCode = geoipRecord.City.Code
	}
	return record
}

var gijson *GIJSON = &GIJSON{Countries: make(map[string]*Country), Cities: []*City{}}

func readStdin(script string) {
	ipRe := regexp.MustCompile("((((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))|((?:[A-F0-9]{1,4}:){7}[A-F0-9]{1,4}))")
	scanner := bufio.NewScanner(os.Stdin)

	DBBox := rice.MustFindBox("db")
	dbFile, err := DBBox.Open("GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	geoipdb, err := geoip.New(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	L, err := newLuaState(script)
	if err != nil {
		log.Fatal(err)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if ip := ipRe.FindString(line); ip != "" {
			if geoipRecord, err := geoipdb.Lookup(ip); err != nil {
				log.Println(err)
			} else if geoipRecord != nil {
				record := newRecord(geoipRecord, ip, line)
				processRecord(record, L)
			}
		}
	}
}

func processRecord(record *Record, L *lua.LState) {
	var found bool = false

	if L != nil {
		var err error
		if record, err = callScript(L, *record); err != nil {
			log.Fatal(err)
		}
		if record == nil {
			// filtered out
			return
		}
	}

	gijson.Countries[record.CountryCode] = &Country{Name: record.Country}

	for _, c := range gijson.Cities {
		if c.Country == record.Country && c.Name == record.City {
			c.Count++
			found = true
			break
		}
	}
	if !found {
		city := &City{
			Country:   record.Country,
			Name:      record.City,
			Latitude:  record.Latitude,
			Longitude: record.Longitude,
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
	script := flag.String("script", "", "lua script to filter/enrich data")
	laddr := flag.String("addr", "", "Address on which to serve up the map")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *script != "" {
		if _, err := os.Stat(*script); os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/gidata", handleGIData)
	http.Handle("/resources/", http.FileServer(rice.MustFindBox("public").HTTPBox()))
	http.HandleFunc("/", serveIndex(*title))

	/* Get the address from the command line or the environment */
	address := *laddr
	if "" == address {
		address = os.Getenv("GIM_ADDR")
	}
	/* Failing that, use an ephemeral port on loopback */
	if "" == address {
		address = "127.0.0.1:0"
	}
	/* If it's a single port, prepend a localhost */
	if _, err := strconv.Atoi(address); nil == err {
		address = "127.0.0.1:" + address
	}

	go readStdin(*script)

	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		addr := fmt.Sprintf("http://%s", l.Addr())
		log.Printf("Listening on %v\n", addr)
		open.Start(addr)
	}()
	log.Fatal(http.Serve(l, nil))
}
