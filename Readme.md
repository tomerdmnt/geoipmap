Display access logs on a geoip world map using [d3.js](http://d3js.org/)

![screentshot](https://raw.githubusercontent.com/tomerdmnt/geoipmap/master/screenshot.png)

# Examples

geoipmap reads logs from stdin

## nginx accesses

```bash
$ ssh root@server.com "zcat -f /var/log/nginx/access.log.* & tail -n 0 -F /var/log/nginx/access.log" | geoipmap -title "nginx access"
```

## ssh accesses

```bash
$ ssh root@server.com "zcat -f /var/log/auth.log.* & tail -n 0 -F /var/log/auth.log" | geoipmap -title "ssh access"
```

# Install

```bash
$ go get github.com/tomerdmnt/geoipmap
```

# Usage

```
Usage: [geoipmap] [-title <title>] [-h]

    geoipmap reads logs from stdin

  -title="": Optional Title
```

# License

This product includes GeoLite data created by MaxMind, available from 
[http://www.maxmind.com](http://www.maxmind.com)
