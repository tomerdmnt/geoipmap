package geoip

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"
)

var (
	metaMarker            = []byte("\xab\xcd\xefMaxMind.com")
	maxMetaSize           = 128 * 1024
	errNoMetadata         = errors.New("can't find metadata - invalid mmdb file?")
	errNoFormatMajor      = errors.New("binary_format_major_version not found in metadata")
	errInvalidFormatMajor = errors.New("binary_format_major_version is not 2")
	errNoIPVersion        = errors.New("missing IP version")
	errInvalidDatabase    = errors.New("database seems to be corrupted")
	errInvalidIP          = errors.New("invalid IP")
	errNoMoreIP           = errors.New("finished looking at the IP addr without finding a match")
	v4InV6Prefix          = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
)

// GeoIP represents an in-memory database which maps IP addresses,
// either IPv4 or IPv6, to geographical information. You shouldn't
// create multiple GeoIP instances. Instead, create only one and
// share it among the different parts of your application. All
// methods are safe to access from multiple goroutines concurrently.
// Use New or Open to initialize a GeoIP.
type GeoIP struct {
	tree         []byte
	data         []byte
	ipVersion    int
	ipv4Start    int
	recordSize   int // bits
	recordBytes  int // bytes rounded to int
	nodeSize     int // bytes
	nodeSizeEven bool
	recordShift  uint // = recordSize - (recordBytes * 8)
	nodeCount    int
	meta         map[string]interface{}
}

// IPVersion returns the IP version the loaded database provides, either
// 4 or 6.
func (g *GeoIP) IPVersion() int {
	return g.ipVersion
}

// Updated returns the date when the loaded database was built.
func (g *GeoIP) Updated() time.Time {
	if t, ok := g.meta["build_epoch"].(uint64); ok {
		return time.Unix(int64(t), 0)
	}
	return time.Time{}
}

// Lookup returns the geographical information for the given
// IP address. Note that addr can be an IP address or a CIDR.
// Both IPv4 and IPv6 are supported by this method.
func (g *GeoIP) Lookup(addr string) (*Record, error) {
	ip, err := g.parseIP(addr)
	if err != nil {
		return nil, err
	}
	return g.LookupIP(ip)
}

// LookupIP works like Lookup, but accepts a net.IP rather
// than the address as a string.
func (g *GeoIP) LookupIP(ip net.IP) (*Record, error) {
	res, err := g.LookupIPValue(ip)
	if err != nil {
		return nil, err
	}
	return g.resultToRecord(res)
}

// LookupIPValue returns the raw value found in the database
// for the given IP. Note that the type of value might vary
// depending on the IP, but will usually be a map[string]interface{}.
func (g *GeoIP) LookupIPValue(ip net.IP) (interface{}, error) {
	if len(ip) == 0 {
		return nil, errInvalidIP
	}
	start := 0
	ipv4 := ip.To4()
	if ipv4 != nil {
		if g.ipVersion == 4 || g.ipv4Start > 0 {
			ip = ipv4
			start = g.ipv4Start
		}
	} else {
		if g.ipVersion == 4 {
			return nil, fmt.Errorf("can't look up IPv6 %s, database is IPv4", ip.String())
		}
	}
	data := []byte(ip)
	return g.lookupData(ip, data, start)
}

func (g *GeoIP) parseIP(addr string) (net.IP, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		// Try a CIDR
		ip, _, _ = net.ParseCIDR(addr)
		if ip == nil {
			return nil, fmt.Errorf("%q is not a valid IPv4 nor IPv6 address", addr)
		}
	}
	return ip, nil
}

func (g *GeoIP) lookupData(ip net.IP, data []byte, node int) (interface{}, error) {
	ii := 0
	bit := 0
	b := data[0]
	for {
		next := g.decodeNode(node, b&0x80 != 0)
		if next == g.nodeCount {
			// Not found
			return nil, fmt.Errorf("address %s not found", ip)
		}
		if next > g.nodeCount {
			// Found data
			return g.lookupResult(next)
		}
		// next < g.nodeCount, keep iterating
		node = next
		bit++
		b = b << 1
		if bit == 8 {
			bit = 0
			ii++
			if len(data) == ii {
				break
			}
			b = data[ii]
		}
	}
	return node, errNoMoreIP
}

func (g *GeoIP) decodeNode(node int, right bool) int {
	data := g.tree[g.nodeSize*node:]
	if g.nodeSizeEven {
		// Format for e.g. 6 bytes
		// | <------------- node --------------->|
		// | 23 .. 0          |          23 .. 0 |
		if right {
			data = data[g.recordBytes:]
		}
		return int(decodeUint64(data, g.recordBytes))
	}
	// Format for e.g. 7 bytes
	// | <------------- node --------------->|
	// | 23 .. 0 | 27..24 | 27..24 | 23 .. 0 |
	if right {
		// Decode value except the most significant nibble
		val := decodeUint64(data[g.recordBytes+1:], g.recordBytes)
		// MSN is the second nibble in the byte just before
		// the decoded ones.
		val = val | uint64(data[g.recordBytes]&0x0F)<<g.recordShift
		return int(val)
	}
	// Decode value except the most significant nibble
	val := decodeUint64(data, g.recordBytes)
	// Add nibble just after the decoded data as the MSB
	val = val | uint64(data[g.recordBytes]>>4)<<g.recordShift
	return int(val)
}

func (g *GeoIP) lookupResult(p int) (interface{}, error) {
	offset := p - g.nodeCount - 16
	dec := &decoder{g.data, offset}
	return dec.decode()
}

func (g *GeoIP) resultToRecord(val interface{}) (*Record, error) {
	return newRecord(val)
}

// New parses the given database as an io.ReadSeeker and returns a new
// GeoIP. If the database does not have the correct format, an error
// will be returned. See also Open.
func New(r io.ReadSeeker) (*GeoIP, error) {
	return newGeoIP(r)
}

// Open initializes a GeoIP from the database named filename. Note that
// if the database file has the .gz extension (e.g. GeoLite2-City.mmdb.gz),
// it will be automatically decompressed in memory before loading it.
func Open(filename string) (*GeoIP, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if filepath.Ext(filename) == ".gz" {
		r, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return New(bytes.NewReader(data))
	}
	return New(f)
}

func newGeoIP(r io.ReadSeeker) (g *GeoIP, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if e, ok := rec.(error); ok {
				err = e
			} else {
				err = errInvalidDatabase
			}
		}
	}()
	// Seek to the end to find the total size
	total, err := r.Seek(0, os.SEEK_END)
	if err != nil {
		return nil, err
	}
	// Seek to the maximum position where the
	// metadata might start, so we only read 128K
	// initially.
	if _, err := r.Seek(-int64(maxMetaSize), os.SEEK_END); err != nil {
		// File might be smaller than maxMetaSize, seek to start
		if _, err := r.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}
	}
	end, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	metaData, err := findMetadata(end)
	if err != nil {
		return nil, err
	}
	dec := &decoder{data: metaData}
	metaVal, err := dec.decode()
	if err != nil {
		return nil, err
	}
	meta, ok := metaVal.(map[string]interface{})
	if !ok {
		return nil, errInvalidDatabase
	}
	major, ok := meta["binary_format_major_version"].(uint16)
	if !ok {
		return nil, errNoFormatMajor
	}
	if major != 2 {
		return nil, errInvalidFormatMajor
	}
	ipVersion, ok := meta["ip_version"].(uint16)
	if !ok {
		return nil, errNoIPVersion
	}
	if ipVersion != 4 && ipVersion != 6 {
		return nil, fmt.Errorf("invalid IP version %d", ipVersion)
	}
	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}
	recordSize := int(meta["record_size"].(uint16))
	nodeCount := int(meta["node_count"].(uint32))
	// Read tree
	nodeSize := recordSize * 2 / 8
	treeSize := nodeSize * nodeCount
	tree := make([]byte, treeSize)
	if _, err := io.ReadFull(r, tree); err != nil {
		return nil, err
	}
	// Discard 16 null bytes after tree
	if _, err := io.CopyN(ioutil.Discard, r, 16); err != nil {
		return nil, err
	}
	// Read the data
	data := make([]byte, int(total)-(treeSize+16+len(metaData)+len(metaMarker)))
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	recordBytes := recordSize / 8
	geo := &GeoIP{
		tree:         tree,
		data:         data,
		ipVersion:    int(ipVersion),
		recordSize:   recordSize,
		recordBytes:  recordBytes,
		nodeSize:     nodeSize,
		nodeSizeEven: nodeSize%2 == 0,
		recordShift:  uint(recordSize - recordBytes*8),
		nodeCount:    nodeCount,
		meta:         meta,
	}
	if ipVersion == 6 {
		s, err := geo.lookupData(nil, v4InV6Prefix, 0)
		if err == errNoMoreIP {
			if i, ok := s.(int); ok {
				geo.ipv4Start = i
			}
		}
	}
	return geo, nil
}

func findMetadata(data []byte) ([]byte, error) {
	if total := len(data); total > maxMetaSize {
		data = data[total-maxMetaSize:]
	}
	p := bytes.LastIndex(data, metaMarker)
	if p == -1 {
		return nil, errNoMetadata
	}
	p += len(metaMarker)
	return data[p:], nil
}
