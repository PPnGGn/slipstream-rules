# slipstream-rules

Trimmed `geosite.dat` / `geoip.dat` for the Slipstream VPN client.

Upstream is [runetfreedom/russia-v2ray-rules-dat](https://github.com/runetfreedom/russia-v2ray-rules-dat)
(full `geosite.dat` is ~70 MB — `ru-blocked-all` alone is 1.37M domains). A weekly
workflow fetches it, keeps only the categories listed in [`categories/`](categories/),
verifies the result loads in xray-core, and publishes it as a GitHub Release —
unless the trimmed output is byte-identical to the current release, in which case
nothing is published.

## Releases

Each release (`YYYYMMDD-HHMM` tag) has:

| asset | what |
|---|---|
| `geosite.dat` | trimmed, xray format, deterministic bytes |
| `geoip.dat` | trimmed, xray format, deterministic bytes |
| `categories.json` | `{"site": ["geosite:category-ru", …], "ip": ["geoip:ru", …]}` — the tokens actually present, so the app knows which `geosite:`/`geoip:` refs it can keep |
| `sha256sums.txt` | `sha256sum` format (`<hex>  <name>`); the app verifies each asset against it |

The app downloads from `releases/latest/download/…`. It does **not** poll for
updates — it fetches once on first install and again only when the user taps
"Обновить".

## Changing what's included

Edit [`categories/geosite.txt`](categories/geosite.txt) or
[`categories/geoip.txt`](categories/geoip.txt) — one category per line. Both the
bare code (`category-ru`) and the token form (`geosite:category-ru`) are
accepted; `#` starts a comment; `*` on its own line keeps everything. Push to
`main` (or run the workflow manually); the next release picks it up. No client
change needed — the app reads `categories.json`.

## Tools

| cmd | what |
|---|---|
| `geofilter` | reads a `.dat`, keeps allowlisted entries, writes it back, prints the kept tokens as JSON |
| `geocheck` | loads a trimmed `dist/` through xray-core's real geo-data loader — fails if a `.dat` won't parse or a `categories.json` token is missing |
| `geolist` | prints every category in a `.dat` with its entry count (inspection only) |

```
go run ./cmd/geofilter -kind geosite -allow categories/geosite.txt -in geosite.dat -out out.dat
go run ./cmd/geocheck  -dir dist
```

`.dat` entries are flat (v2fly's compiler expands `include:` upstream), so
filtering by name never breaks a dependency.

## Repo setup

- **Visibility: Public** (the app downloads release assets unauthenticated).
- Settings → Actions → General → Workflow permissions → **Read and write**.
- No secrets — `GITHUB_TOKEN` covers `gh release create` and the failure issue.
- A scheduled workflow is disabled by GitHub after ~60 days without repo
  activity; any push re-arms it. GitHub emails the workflow's last editor when a
  scheduled run fails, and the run also opens/updates a "build geo failed" issue.

## License

Code (`cmd/`, workflow) and the published `.dat` files are **GPL-3.0** — the geo
data is derived from `runetfreedom/russia-v2ray-rules-dat`, which is GPL-3.0, so
anything redistributing it inherits those terms. See [`LICENSE`](LICENSE).

The Slipstream app merely downloads and reads these files at runtime (it doesn't
link or bundle them), which is use, not redistribution — the app's own license is
unaffected.
