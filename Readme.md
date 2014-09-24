view ip accesses to your server on a world map using [d3.js](http://d3js.org/)

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

## fail2ban
```bash
$ ssh root@server.com "zcat -f /var/log/fail2ban.log* & tail -n 0 -F /var/log/fail2ban.log" | grep Ban | geoipmap -title "fail2ban"
```

# Install

Download [binaries](https://github.com/tomerdmnt/geoipmap/releases)

or
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
