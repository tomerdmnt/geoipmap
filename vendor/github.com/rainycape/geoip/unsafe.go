// +build !appengine

package geoip

import (
	"unsafe"
)

func makeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
