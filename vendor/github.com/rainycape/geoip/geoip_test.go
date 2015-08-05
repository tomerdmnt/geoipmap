package geoip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func readFile(t testing.TB, filename string) []byte {
	t.Logf("opening file %s", filename)
	p := filepath.Join("testdata", filepath.FromSlash(filename))
	data, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("missing file %s", p)
		}
		t.Fatal(err)
	}
	return data
}

func testNewGeoIP(t testing.TB, filename string) *GeoIP {
	data := readFile(t, filename)
	geo, err := New(bytes.NewReader(data))
	if err != nil {
		t.Error(err)
		return nil
	}
	return geo
}

func TestGeoIP(t *testing.T) {
	jsonData := readFile(t, "source-data/GeoIP2-City-Test.json")
	lookups := make(map[string]interface{})
	for _, v := range bytes.Split(jsonData, []byte("\n")) {
		if len(v) == 0 {
			continue
		}
		var out []interface{}
		dec := json.NewDecoder(bytes.NewReader(v))
		if err := dec.Decode(&out); err != nil {
			t.Fatal(err)
		}
		lookups[out[0].(string)] = normalize(out[1])
	}
	// Test against all dbs that should correctly return the data
	geo := testNewGeoIP(t, "GeoIP2-City-Test.mmdb")
	if geo != nil {
		testLookups(t, geo, lookups)
	}
}

func toNumber(v float64) interface{} {
	if math.Ceil(v) == v {
		return int(v)
	}
	return v
}

func normalize(val interface{}) interface{} {
	switch x := val.(type) {
	case map[string]interface{}:
		var keys []string
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m := make(map[string]interface{}, len(x))
		for _, k := range keys {
			v := x[k]
			// The control JSON has these keys as strings, convert them to float
			if k == "latitude" || k == "longitude" {
				if s, ok := v.(string); ok {
					v, _ = strconv.ParseFloat(s, 64)
				}
			}
			m[k] = normalize(v)
		}
		return m
	case []interface{}:
		for ii, v := range x {
			x[ii] = normalize(v)
		}
	case string:
		return x
	case float64:
		return toNumber(x)
	case uint32:
		return toNumber(float64(x))
	case uint16:
		return toNumber(float64(x))
	case bool:
		if x {
			return int(1)
		}
		return int(0)
	default:
		panic(fmt.Sprintf("can't normalize %T", val))
	}
	return val
}

func testLookups(t testing.TB, g *GeoIP, lookups map[string]interface{}) {
	for k, v := range lookups {
		ip := net.ParseIP(k)
		if ip == nil {
			ip, _, _ = net.ParseCIDR(k)
		}
		val, err := g.LookupIPValue(ip)
		if err != nil {
			t.Error(err)
			continue
		}
		res := normalize(val)
		s1 := fmt.Sprintf("%v", v)
		s2 := fmt.Sprintf("%v", res)
		if s1 != s2 {
			t.Errorf("expecting %v for ip %q, got %v instead", s1, k, s2)
		}
	}
}

func TestTestDatabases(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("testdata", "MaxMind-DB*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range matches {
		name := filepath.Base(v)
		geo := testNewGeoIP(t, name)
		if geo == nil {
			continue
		}
		testIPv4(t, geo, name)
		if geo.IPVersion() == 6 {
			testIPv6(t, geo, name)
		}
	}
}

func testIPv4(t testing.TB, g *GeoIP, name string) {
	for ii := 1; ii < 36; ii++ {
		address := fmt.Sprintf("1.1.1.%d", ii)
		ip := net.ParseIP(address)
		res, err := g.LookupIPValue(ip)
		expected := expectedValue(ip, name)
		if expected == nil {
			if err == nil {
				t.Errorf("expecting an error for IP %s in DB %s", ip, name)
			}
		} else {
			if err != nil {
				t.Error(err)
			} else {
				if !deepEqual(t, res, expected) {
					t.Errorf("expecting %v for IP %s in DB %s, got %v instead", expected, ip, name, res)
				}
			}
		}
	}
}

func testIPv6(t testing.TB, g *GeoIP, name string) {
	addrs := []string{
		"::1:ffff:ffff",
		"::2:0:0",
		"::2:0:40",
		"::2:0:50",
		"::2:0:58",
	}
	for _, v := range addrs {
		ip := net.ParseIP(v)
		res, err := g.LookupIPValue(ip)
		expected := expectedValue(ip, name)
		if expected == nil {
			if err == nil {
				t.Errorf("expecting an error for IP %s in DB %s", ip, name)
			}
		} else {
			if err != nil {
				t.Error(err)
			} else {
				if !deepEqual(t, res, expected) {
					t.Errorf("expecting %v for IP %s in DB %s, got %v instead", expected, ip, name, res)
				}
			}
		}
	}
}

func deepEqual(t testing.TB, a interface{}, b interface{}) bool {
	if !reflect.DeepEqual(a, b) {
		ma, aok := a.(map[string]interface{})
		mb, bok := b.(map[string]interface{})
		if aok && bok {
			for k, v := range ma {
				vb := mb[k]
				if !reflect.DeepEqual(v, vb) {
					t.Logf("key %q differs: %v and %v", k, v, vb)
				}
			}
		}
		return false
	}
	return true
}

func prev2Power(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	v := ip[3]
	ip[3] = byte(math.Pow(2, math.Floor(math.Log2(float64(ip[3])))))
	s := ip.String()
	ip[3] = v
	return s
}

func expectedValue(ip net.IP, name string) interface{} {
	switch name {
	case "MaxMind-DB-no-ipv4-search-tree.mmdb":
		return "::0/64"
	case "MaxMind-DB-string-value-entries.mmdb":
		if v4 := ip.To4(); v4 != nil {
			if v4[3] > 32 {
				return nil
			}
			s := prev2Power(ip)
			last := s[strings.LastIndex(s, ".")+1:]
			switch last {
			case "1", "32":
				s += "/32"
			case "2":
				s += "/31"
			case "4":
				s += "/30"
			case "8":
				s += "/29"
			case "16":
				s += "/28"
			}
			return s
		}
	case "MaxMind-DB-test-broken-pointers-24.mmdb":
		if v4 := ip.To4(); v4 != nil {
			if v4[3] < 16 {
				return map[string]interface{}{"ip": prev2Power(ip)}
			}
		}
	case "MaxMind-DB-test-ipv4-24.mmdb", "MaxMind-DB-test-ipv4-28.mmdb", "MaxMind-DB-test-ipv4-32.mmdb":
		if v4 := ip.To4(); v4 != nil {
			if v4[3] > 32 {
				return nil
			}
			return map[string]interface{}{"ip": prev2Power(ip)}
		}
	case "MaxMind-DB-test-ipv6-24.mmdb", "MaxMind-DB-test-ipv6-28.mmdb", "MaxMind-DB-test-ipv6-32.mmdb":
		if ip.To4() != nil {
			return nil
		}
		return map[string]interface{}{"ip": ip.String()}
	case "MaxMind-DB-test-mixed-24.mmdb", "MaxMind-DB-test-mixed-28.mmdb", "MaxMind-DB-test-mixed-32.mmdb":
		if v4 := ip.To4(); v4 != nil {
			if v4[3] > 32 {
				return nil
			}
			return map[string]interface{}{"ip": "::" + prev2Power(ip)}
		}
		return map[string]interface{}{"ip": ip.String()}
	case "MaxMind-DB-test-nested.mmdb":
		if ip.To4() == nil {
			return nil
		}
		return map[string]interface{}{
			"map1": map[string]interface{}{
				"map2": map[string]interface{}{
					"array": []interface{}{
						map[string]interface{}{
							"map3": map[string]interface{}{
								"a": uint32(1),
								"b": uint32(2),
								"c": uint32(3),
							},
						},
					},
				},
			},
		}
	case "MaxMind-DB-test-decoder.mmdb":
		if ip.To4() == nil {
			return nil
		}
		return map[string]interface{}{
			"boolean": true,
			"float":   float32(1.1),
			"int32":   int32(-268435456),
			"map": map[string]interface{}{
				"mapX": map[string]interface{}{
					"arrayX":       []interface{}{uint32(7), uint32(8), uint32(9)},
					"utf8_stringX": "hello",
				},
			},
			"uint64":      uint64(1152921504606846976),
			"array":       []interface{}{uint32(1), uint32(2), uint32(3)},
			"bytes":       []byte{0, 0, 0, 42},
			"double":      float64(42.123456),
			"uint128":     makeBigInt("12194330274671844653834364178879555882988077825061979814999074790014971526343581141333713244905439924544851208311796974731898472801654660045227"),
			"uint16":      uint16(100),
			"uint32":      uint32(268435456),
			"utf8_string": "unicode! \u262f - \u266b",
		}
	}
	return nil
}

func TestOpenGz(t *testing.T) {
	file := filepath.Join("testdata", "GeoIP2-City-Test.mmdb.gz")
	_, err := Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("missing file %s", file)
		}
		t.Fatal(err)
	}
}

func makeBigInt(s string) *big.Int {
	i := new(big.Int)
	if _, err := fmt.Sscan(s, i); err != nil {
		panic(err)
	}
	return i
}
