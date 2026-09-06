# slipstream-rules

Trimmed `geosite.dat` / `geoip.dat` for the Slipstream VPN client.

Upstream is [runetfreedom/russia-v2ray-rules-dat](https://github.com/runetfreedom/russia-v2ray-rules-dat)
(full `geosite.dat` is ~70 MB — `ru-blocked-all` alone is 1.37M domains). A daily
workflow fetches it, keeps only the categories listed in [`categories/`](categories/),
and publishes the result as a GitHub Release.

## Releases

Each release (`YYYYMMDD-HHMM` tag) has:

| asset | what |
|---|---|
| `geosite.dat` | trimmed, xray format |
| `geoip.dat` | trimmed, xray format |
| `categories.json` | `{"site": ["geosite:category-ru", …], "ip": ["geoip:ru", …]}` — the tokens actually present, so the app knows which `geosite:`/`geoip:` refs it can keep |
| `sha256sums.txt` | integrity |

The app downloads from `releases/latest/download/…`.

## Changing what's included

Edit [`categories/geosite.txt`](categories/geosite.txt) or
[`categories/geoip.txt`](categories/geoip.txt) — one category per line, written
as it appears in a config (`geosite:category-ru` → `category-ru`). `*` on its own
line keeps everything. Push to `main` (or run the workflow manually); the next
release picks it up. No client change needed — the app reads `categories.json`.

## geofilter

`cmd/geofilter` reads a `.dat`, keeps entries whose category is allowlisted,
writes it back, and prints the kept tokens as JSON:

```
go run ./cmd/geofilter -kind geosite -allow categories/geosite.txt -in geosite.dat -out out.dat
```

`.dat` entries are flat (v2fly's compiler expands `include:` upstream), so
filtering by name never breaks a dependency.

## Repo setup

- **Visibility: Public** (the app downloads release assets unauthenticated).
- Settings → Actions → General → Workflow permissions → **Read and write**.
- No secrets — `GITHUB_TOKEN` is enough for `gh release create`.
