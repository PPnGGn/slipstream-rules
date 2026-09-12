// Command geocheck verifies a freshly trimmed dist/ directory: every token in
// categories.json must build a real matcher through xray-core's geo-data
// loader — the same code path config.Build() runs. This catches a .dat that
// xray can't parse (e.g. after an xray-core bump), a categories.json token
// that isn't actually present in the trimmed file, and a corrupt individual
// entry inside a category that is present.
//
//	geocheck -dir dist
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xtls/xray-core/common/geodata"
	"google.golang.org/protobuf/proto"
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

	// ParseDomainRules/ParseIPRules alone only run geodata's checkFile — a
	// linear scan confirming the category's record is present in the .dat.
	// Building the real matcher (DomainReg/IPReg, the registries config.Build()
	// uses) additionally decodes every domain/CIDR entry inside that record,
	// so a corrupt individual entry fails here instead of surfacing on a
	// user's device.
	domainRules, err := geodata.ParseDomainRules(cat.Site, geodata.Domain_Full)
	if err != nil {
		fail("geosite token did not resolve against %s/geosite.dat: %v", *dir, err)
	}
	for i, token := range cat.Site {
		if _, err := geodata.DomainReg.BuildDomainMatcher(domainRules[i : i+1]); err != nil {
			fail("geosite token %q failed to build a matcher: %v", token, err)
		}
	}

	ipRules, err := geodata.ParseIPRules(cat.IP)
	if err != nil {
		fail("geoip token did not resolve against %s/geoip.dat: %v", *dir, err)
	}
	for i, token := range cat.IP {
		if _, err := geodata.IPReg.BuildIPMatcher(ipRules[i : i+1]); err != nil {
			fail("geoip token %q failed to build a matcher: %v", token, err)
		}
	}

	printEntryCounts(*dir, cat.Site, cat.IP)

	fmt.Printf("geocheck: %d geosite + %d geoip tokens resolve and build against %s\n",
		len(cat.Site), len(cat.IP), *dir)
}

// printEntryCounts logs the domain/CIDR count behind each kept category, so a
// CI log flags an upstream category that suddenly ballooned (e.g. RunetFreedom
// folding a much larger list into category-ru) even though the build itself
// still succeeds.
func printEntryCounts(dir string, siteTokens, ipTokens []string) {
	siteCounts := map[string]int{}
	if raw, err := os.ReadFile(filepath.Join(dir, "geosite.dat")); err == nil {
		var l geodata.GeoSiteList
		if proto.Unmarshal(raw, &l) == nil {
			for _, e := range l.Entry {
				siteCounts[strings.ToLower(e.Code)] = len(e.Domain)
			}
		}
	}
	ipCounts := map[string]int{}
	if raw, err := os.ReadFile(filepath.Join(dir, "geoip.dat")); err == nil {
		var l geodata.GeoIPList
		if proto.Unmarshal(raw, &l) == nil {
			for _, e := range l.Entry {
				ipCounts[strings.ToLower(e.Code)] = len(e.Cidr)
			}
		}
	}

	type row struct {
		token string
		n     int
	}
	var rows []row
	for _, t := range siteTokens {
		rows = append(rows, row{t, siteCounts[strings.TrimPrefix(t, "geosite:")]})
	}
	for _, t := range ipTokens {
		rows = append(rows, row{t, ipCounts[strings.TrimPrefix(t, "geoip:")]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].token < rows[j].token })
	for _, r := range rows {
		fmt.Fprintf(os.Stderr, "  %-32s %d entries\n", r.token, r.n)
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "geocheck: "+format+"\n", a...)
	os.Exit(1)
}
