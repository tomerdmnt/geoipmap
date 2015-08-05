geoip
=====

GeoIP2 library in Go (golang)

This library implements reading and decoding of GeoIP2 databases. Free
databases can be downloaded from [MaxMind][1]. 

To install geoip run the following command:

```
    go get gopkgs.com/geoip.v1
```

Then use the following import path to ensure a stable API:

```go
    import "gopkgs.com/geoip.v1"
```

For documentation and available versions,
see http://gopkgs.com/geoip.


## Example

```go
package main

import (
	"fmt"
	"gopkgs.com/geoip.v1"
)

func main() {
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
```

## License

This code is licensed under the [MPL 2.0][2].

[1]: http://dev.maxmind.com/geoip/geoip2/geolite2/
[2]: http://www.mozilla.org/MPL/2.0/
