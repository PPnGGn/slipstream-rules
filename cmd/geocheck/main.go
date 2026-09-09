// Command geocheck verifies a freshly trimmed dist/ directory: every token in
// categories.json must resolve through xray-core's real geo-data loader — the
// same code path config.Build() runs. This catches a .dat that xray can't
// parse (e.g. after an xray-core bump) and a categories.json token that isn't
// actually present in the trimmed file.
//
//	geocheck -dir dist
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xtls/xray-core/common/geodata"
)

func main() {
	dir := flag.String("dir", "dist", "directory holding geosite.dat, geoip.dat, categories.json")
	flag.Parse()

	// geodata.ParseDomainRules("geosite:x") -> filesystem.OpenAsset("geosite.dat")
	// -> platform.GetAssetLocation reads this env. Same as slipstream-core.
	os.Setenv("xray.location.asset", *dir)

	raw, err := os.ReadFile(filepath.Join(*dir, "categories.json"))
	if err != nil {
		fail("read categories.json: %v", err)
	}
	var cat struct {
		Site []string `json:"site"`
		IP   []string `json:"ip"`
	}
	if err := json.Unmarshal(raw, &cat); err != nil {
		fail("parse categories.json: %v", err)
	}
	if len(cat.Site) == 0 || len(cat.IP) == 0 {
		fail("categories.json has an empty site or ip list (site=%d ip=%d)", len(cat.Site), len(cat.IP))
	}

	if _, err := geodata.ParseDomainRules(cat.Site, geodata.Domain_Full); err != nil {
		fail("geosite token did not resolve against %s/geosite.dat: %v", *dir, err)
	}
	if _, err := geodata.ParseIPRules(cat.IP); err != nil {
		fail("geoip token did not resolve against %s/geoip.dat: %v", *dir, err)
	}

	fmt.Printf("geocheck: %d geosite + %d geoip tokens resolve against %s\n",
		len(cat.Site), len(cat.IP), *dir)
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "geocheck: "+format+"\n", a...)
	os.Exit(1)
}
