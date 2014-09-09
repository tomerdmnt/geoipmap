Display access logs on a geoip world map

# Examples

## nginx access:

```bash
$ ssh root@server.com "zcat -f /var/log/nginx/access.log.* <(tail -n 0 -F /var/log/nginx/access.log)" | geoipmap
```

## ssh access

```bash
$ ssh root@server.com "zcat -f /var/log/auth.log.* <(tail -n 0 -F /var/log/auth.log)" | geoipmap
```

# Install

```bash
$ go get github.com/tomerdmnt/geoipmap
```

# License

This product includes GeoLite data created by MaxMind, available from 
[http://www.maxmind.com](http://www.maxmind.com)
