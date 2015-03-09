view ip accesses to your server on a world map using [d3.js](http://d3js.org/)

![screentshot](https://raw.githubusercontent.com/tomerdmnt/geoipmap/master/screenshot.png)

# Examples

geoipmap reads logs from stdin and displays the geo ip data on a world map

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

## Use Lua scripts

Most filtering can be done using bash tools, but you can also use the -script flag to process records using Lua

script.lua:

```lua
function record(r)
    if r.Country == "Greenland" then
        -- filter out ips from greenland
        return false
    end

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
```

```bash
 $ ... | geoipmap -script script.lua
```

# Install

Download [binaries](https://github.com/tomerdmnt/geoipmap/releases)

or
```bash
$ go get github.com/tomerdmnt/geoipmap
```

# License

This product includes GeoLite data created by MaxMind, available from 
[http://www.maxmind.com](http://www.maxmind.com)
