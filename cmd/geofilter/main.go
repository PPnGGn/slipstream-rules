// Command geofilter trims a v2ray/xray geo database (geosite.dat or geoip.dat)
// down to an allowlist of categories.
//
// It reads a .dat file, keeps only the entries whose category code appears in
// the allowlist, writes the trimmed .dat back out, and prints the kept category
// tokens (e.g. "geosite:category-ru") to stdout as a JSON array — the build
// workflow merges those into categories.json.
//
//	geofilter -kind geosite -allow categories/geosite.txt -in raw/geosite.dat -out dist/geosite.dat
//	geofilter -kind geoip   -allow categories/geoip.txt   -in raw/geoip.dat   -out dist/geoip.dat
//
// Entries in a .dat are already flat (v2fly's compiler expands `include:` at
// build time), so filtering by category name never breaks a dependency.
package main

import (
	"bufio"
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
	kind := flag.String("kind", "", "geosite | geoip")
	allowPath := flag.String("allow", "", "allowlist file (one category per line, # comments)")
	inPath := flag.String("in", "", "input .dat")
	outPath := flag.String("out", "", "output .dat")
	flag.Parse()

	if *kind != "geosite" && *kind != "geoip" {
		fail("-kind must be geosite or geoip")
	}
	if *allowPath == "" || *inPath == "" || *outPath == "" {
		fail("-allow, -in and -out are required")
	}

	allow, keepAll, err := readAllowlist(*allowPath)
	if err != nil {
		fail("allowlist: %v", err)
	}
	raw, err := os.ReadFile(*inPath)
	if err != nil {
		fail("read %s: %v", *inPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fail("mkdir: %v", err)
	}

	keep := func(code string) bool { return keepAll || allow[strings.ToUpper(code)] }

	var kept []string
	switch *kind {
	case "geosite":
		kept, err = filterGeosite(raw, keep, *outPath)
	case "geoip":
		kept, err = filterGeoip(raw, keep, *outPath)
	}
	if err != nil {
		fail("%v", err)
	}

	// Warn (don't fail) about allowlist entries that weren't in the source —
	// a typo or a category RunetFreedom renamed.
	if !keepAll {
		keptSet := make(map[string]bool, len(kept))
		for _, k := range kept {
			keptSet[k] = true
		}
		for code := range allow {
			token := *kind + ":" + strings.ToLower(code)
			if !keptSet[token] {
				fmt.Fprintf(os.Stderr, "warning: %q from allowlist not found in %s\n", token, *inPath)
			}
		}
	}

	sort.Strings(kept)
	out, _ := json.Marshal(kept)
	fmt.Println(string(out))

	outSize := int64(-1)
	if fi, err := os.Stat(*outPath); err == nil {
		outSize = fi.Size()
	}
	fmt.Fprintf(os.Stderr, "%s: kept %d categories, %dKB -> %dKB\n",
		*kind, len(kept), int64(len(raw))>>10, outSize>>10)
}

func filterGeosite(raw []byte, keep func(string) bool, outPath string) ([]string, error) {
	var list geodata.GeoSiteList
	if err := proto.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("unmarshal geosite: %w", err)
	}
	trimmed := &geodata.GeoSiteList{}
	var kept []string
	for _, e := range list.Entry {
		if keep(e.Code) {
			trimmed.Entry = append(trimmed.Entry, e)
			kept = append(kept, "geosite:"+strings.ToLower(e.Code))
		}
	}
	return kept, marshalTo(trimmed, outPath)
}

func filterGeoip(raw []byte, keep func(string) bool, outPath string) ([]string, error) {
	var list geodata.GeoIPList
	if err := proto.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("unmarshal geoip: %w", err)
	}
	trimmed := &geodata.GeoIPList{}
	var kept []string
	for _, e := range list.Entry {
		if keep(e.Code) {
			trimmed.Entry = append(trimmed.Entry, e)
			kept = append(kept, "geoip:"+strings.ToLower(e.Code))
		}
	}
	return kept, marshalTo(trimmed, outPath)
}

func marshalTo(m proto.Message, outPath string) error {
	// Deterministic: identical input -> identical bytes -> identical sha256, so
	// the workflow can skip publishing a release when nothing actually changed.
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(outPath, b, 0o644)
}

// readAllowlist returns a set of UPPERCASE category codes. Lines may be written
// either bare ("category-ru") or with the token prefix ("geosite:category-ru" /
// "geoip:ru") — both forms resolve to the same code. Blank lines and #-comments
// (whole-line or trailing) are ignored. A lone "*" means keep every category
// (keepAll = true). Duplicate entries are warned about, not an error.
func readAllowlist(path string) (set map[string]bool, keepAll bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	set = make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "*" {
			keepAll = true
			continue
		}
		line = strings.TrimPrefix(line, "geosite:")
		line = strings.TrimPrefix(line, "geoip:")
		code := strings.ToUpper(line)
		if set[code] {
			fmt.Fprintf(os.Stderr, "warning: duplicate allowlist entry %q\n", code)
		}
		set[code] = true
	}
	return set, keepAll, sc.Err()
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "geofilter: "+format+"\n", a...)
	os.Exit(1)
}
