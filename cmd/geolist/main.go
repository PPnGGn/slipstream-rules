// geolist: prints every category in a geosite.dat / geoip.dat with its entry count.
//
//	go run ./cmd/geolist -kind geosite -in geosite.dat
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/xtls/xray-core/common/geodata"
	"google.golang.org/protobuf/proto"
)

func main() {
	kind := flag.String("kind", "geosite", "geosite | geoip")
	in := flag.String("in", "", "input .dat")
	flag.Parse()
	raw, err := os.ReadFile(*in)
	if err != nil {
		panic(err)
	}
	type row struct {
		code string
		n    int
	}
	var rows []row
	if *kind == "geosite" {
		var l geodata.GeoSiteList
		if err := proto.Unmarshal(raw, &l); err != nil {
			panic(err)
		}
		for _, e := range l.Entry {
			rows = append(rows, row{e.Code, len(e.Domain)})
		}
	} else {
		var l geodata.GeoIPList
		if err := proto.Unmarshal(raw, &l); err != nil {
			panic(err)
		}
		for _, e := range l.Entry {
			rows = append(rows, row{e.Code, len(e.Cidr)})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].code < rows[j].code })
	for _, r := range rows {
		fmt.Printf("%-28s %d\n", r.code, r.n)
	}
	fmt.Fprintf(os.Stderr, "total: %d categories\n", len(rows))
}
