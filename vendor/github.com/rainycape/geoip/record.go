package geoip

import (
	"fmt"
)

var (
	codes = []string{"iso_code", "code"}
)

// Name represents a name with multiple localized names.
type Name map[string]string

func (n Name) String() string {
	return n["en"]
}

// LocalizedName returns the name in the given language, or
// the empty string if the name lacks that translation.
func (n Name) LocalizedName(lang string) string {
	return n[lang]
}

// Localizations returns the available localizations for
// this name.
func (n Name) Localizations() []string {
	keys := make([]string, 0, len(n))
	for k := range n {
		keys = append(keys, k)
	}
	return keys
}

// Place represents a place with a Code, a GeonameID
// (see http://www.geonames.org for more information), and a Name.
// Code and GeonameID might be empty, but Name will always have at
// least a value.
type Place struct {
	// Code is the given code for the place. For continents, this
	// value is one of AF (Africa), AS (Asia), EU (Europe), OC (Oceania),
	// NA (North America) and SA (South America). For countries, its
	// their ISO 3166-1 2 letter code (see http://en.wikipedia.org/wiki/ISO_3166-1).
	Code string
	// GeonameID is the place's ID in the geonames database. See
	// http://www.geonames.org for more information.
	GeonameID int
	// Name is the place name, usually with several translations.
	Name Name
}

func (p *Place) String() string {
	return p.Name.String()
}

// Record hold the information returned for a given
// IP address. See the comments on each field for more
// information.
type Record struct {
	// Continent contains information about the continent
	// where the record is located.
	Continent *Place
	// Country contains information about the country
	// where the record is located.
	Country *Place
	// RegisteredCountry contains information about the
	// country where the ISP has registered the IP address
	// for this record. Note that this field might be
	// different from Country.
	RegisteredCountry *Place
	// RepresentedCountry is non nil only when the record
	// belongs an entity representing a country, like an
	// embassy or a military base. Note that it might be
	// diferrent from Country.
	RepresentedCountry *Place
	// City contains information about the city where the
	// record is located.
	City *Place
	// Subdivisions contains details about the subdivisions
	// of the country where the record is located. Subdivisions
	// are arranged from largest to smallest and the number of
	// them will vary depending on the country.
	Subdivisions []*Place
	// Latitude of the location associated with the record.
	// Note that a 0 Latitude and a 0 Longitude means the
	// coordinates are not known.
	Latitude float64
	// Longitude of the location associated with the record.
	// Note that a 0 Latitude and a 0 Longitude means the
	// coordinates are not known.
	Longitude float64
	// MetroCode contains the metro code associated with the
	// record. These are only available in the US
	MetroCode int
	// PostalCode associated with the record. These are available in
	// AU, CA, FR, DE, IT, ES, CH, UK and US.
	PostalCode string
	// TimeZone associated with the record, in IANA format (e.g.
	// America/New_York). See http://www.iana.org/time-zones.
	TimeZone string
	// IsAnonymousProxy is true iff the record belongs
	// to an anonymous proxy.
	IsAnonymousProxy bool
	// IsSatelliteProvider is true iff the record is
	// in a block managed by a satellite ISP that provides
	// service to multiple countries. These IPs might be
	// in high risk countries.
	IsSatelliteProvider bool
}

// CountryCode is a shorthand for r.Country.Code, but returns
// the empty string if r.Country is nil.
func (r *Record) CountryCode() string {
	if r != nil && r.Country != nil {
		return r.Country.Code
	}
	return ""
}

func newPlace(val interface{}) *Place {
	if m, ok := val.(map[string]interface{}); ok {
		geonameId := int(m["geoname_id"].(uint32))
		var code string
		for _, v := range codes {
			if c, ok := m[v].(string); ok {
				code = c
				break
			}
		}
		var name Name
		if names, ok := m["names"].(map[string]interface{}); ok {
			name = make(Name, len(names))
			for k, v := range names {
				if n, ok := v.(string); ok {
					name[k] = n
				}
			}
		}
		return &Place{
			Code:      code,
			GeonameID: geonameId,
			Name:      name,
		}
	}
	return nil
}

func newRecord(val interface{}) (*Record, error) {
	m, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid record type %T", val)
	}
	var latitude, longitude float64
	var postalCode, timeZone string
	var metroCode int
	var isAnonymousProxy, isSatelliteProvider bool
	if location, ok := m["location"].(map[string]interface{}); ok {
		latitude, _ = location["latitude"].(float64)
		longitude, _ = location["longitude"].(float64)
		if m := location["metro_code"]; m != nil {
			metroCode = int(m.(uint16))
		}
		timeZone, _ = location["time_zone"].(string)
	}
	if postal, ok := m["postal"].(map[string]interface{}); ok {
		postalCode, _ = postal["code"].(string)
	}
	var subdivisions []*Place
	if subs, ok := m["subdivisions"].([]interface{}); ok {
		for _, v := range subs {
			if p := newPlace(v); p != nil {
				subdivisions = append(subdivisions, p)
			}
		}
	}
	if traits, ok := m["traits"].(map[string]interface{}); ok {
		isAnonymousProxy, _ = traits["is_anonymous_proxy"].(bool)
		isSatelliteProvider, _ = traits["is_satellite_provider"].(bool)
	}
	return &Record{
		Continent:           newPlace(m["continent"]),
		Country:             newPlace(m["country"]),
		RegisteredCountry:   newPlace(m["registered_country"]),
		RepresentedCountry:  newPlace(m["represented_country"]),
		City:                newPlace(m["city"]),
		Subdivisions:        subdivisions,
		Latitude:            latitude,
		Longitude:           longitude,
		MetroCode:           metroCode,
		PostalCode:          postalCode,
		TimeZone:            timeZone,
		IsAnonymousProxy:    isAnonymousProxy,
		IsSatelliteProvider: isSatelliteProvider,
	}, nil
}
