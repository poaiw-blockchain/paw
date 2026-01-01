# GeoIP Database Configuration (SEC-3.7)

PAW uses geographic diversity to prevent regional validator concentration. See `x/oracle/keeper/geoip.go`.

## Required: MaxMind GeoLite2-Country.mmdb

Download (free, registration required): https://dev.maxmind.com/geoip/geolite2-free-geolocation-data

## Installation Locations (checked in order)

1. `/usr/share/GeoIP/GeoLite2-Country.mmdb`
2. `/var/lib/GeoIP/GeoLite2-Country.mmdb`
3. `./GeoLite2-Country.mmdb` (working directory)
4. `GEOIP_DB_PATH` environment variable

## Quick Setup

```bash
export GEOIP_DB_PATH=/path/to/GeoLite2-Country.mmdb
# OR
sudo cp GeoLite2-Country.mmdb /var/lib/GeoIP/
```

## Cache: LRU with 1000 entries, 1 hour TTL (governance-configurable)

## Update: Weekly via MaxMind's `geoipupdate` tool

## Fallback
- Missing database: validator registration fails
- Private IPs: rejected in production
- Unknown IPs: "unknown" region (testnet only)

## Security
- Local database lookups only (no external API)
- Deterministic across nodes for consensus compatibility
