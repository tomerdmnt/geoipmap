SRC=$(wildcard *.go)

realse: geoipmap-windows-amd64.tar.gz geoipmap-darwin-amd64.tar.gz geoipmap-linux-amd64.tar.gz

geoipmap-windows-amd64.tar.gz: $(SRC)
	GOOS=windows go build
	rice append --exec geoipmap.exe
	tar cfz $@ geoipmap.exe

geoipmap-darwin-amd64.tar.gz: $(SRC)
	GOOS=darwin go build
	rice append --exec geoipmap
	tar cfz $@ geoipmap

geoipmap-linux-amd64.tar.gz: $(SRC)
	GOOS=linux go build
	rice append --exec geoipmap
	tar cfz $@ geoipmap

.PHONY: release
