package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "allow.txt")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadAllowlist(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		want         map[string]bool
		wantOptional map[string]bool
		keepAll      bool
	}{
		{
			name: "comments, blanks and trailing # are ignored",
			body: "# header\n\ncategory-ru\n  vk   # keep\n\n# tail\n",
			want: map[string]bool{"CATEGORY-RU": true, "VK": true},
		},
		{
			name:    "lone star means keep everything",
			body:    "# geoip\n*\nprivate\n",
			want:    map[string]bool{"PRIVATE": true},
			keepAll: true,
		},
		{
			name: "token prefixes are accepted and normalised",
			body: "geosite:category-ru\ngeoip:ru\nCategory-Gov-RU\n",
			want: map[string]bool{"CATEGORY-RU": true, "RU": true, "CATEGORY-GOV-RU": true},
		},
		{
			name: "duplicates collapse without error",
			body: "vk\nVK\ngeosite:vk\n",
			want: map[string]bool{"VK": true},
		},
		{
			name:         "leading ? marks an entry optional and is stripped from the code",
			body:         "category-ru\n?category-ads-ir\n?geosite:category-ads-de\n",
			want:         map[string]bool{"CATEGORY-RU": true, "CATEGORY-ADS-IR": true, "CATEGORY-ADS-DE": true},
			wantOptional: map[string]bool{"CATEGORY-ADS-IR": true, "CATEGORY-ADS-DE": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			set, optional, keepAll, err := readAllowlist(writeTemp(t, tc.body))
			if err != nil {
				t.Fatalf("readAllowlist: %v", err)
			}
			if keepAll != tc.keepAll {
				t.Errorf("keepAll = %v, want %v", keepAll, tc.keepAll)
			}
			if !reflect.DeepEqual(set, tc.want) {
				t.Errorf("set = %v, want %v", set, tc.want)
			}
			wantOptional := tc.wantOptional
			if wantOptional == nil {
				wantOptional = map[string]bool{}
			}
			if !reflect.DeepEqual(optional, wantOptional) {
				t.Errorf("optional = %v, want %v", optional, wantOptional)
			}
		})
	}
}

func TestReadAllowlistMissingFile(t *testing.T) {
	if _, _, _, err := readAllowlist(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestCheckAllowlistCoverage(t *testing.T) {
	t.Run("required entry missing from the source -> reported, no warning needed", func(t *testing.T) {
		allow := map[string]bool{"CATEGORY-RU": true, "VK": true}
		optional := map[string]bool{}
		kept := []string{"geosite:vk"} // category-ru didn't make it into the trimmed .dat
		var warnBuf bytes.Buffer

		missing := checkAllowlistCoverage("geosite", allow, optional, kept, &warnBuf)

		if !reflect.DeepEqual(missing, []string{"geosite:category-ru"}) {
			t.Errorf("missing = %v, want [geosite:category-ru]", missing)
		}
	})

	t.Run("optional entry missing -> warns, not reported as missing", func(t *testing.T) {
		allow := map[string]bool{"CATEGORY-RU": true, "CATEGORY-ADS-IR": true}
		optional := map[string]bool{"CATEGORY-ADS-IR": true}
		kept := []string{"geosite:category-ru"}
		var warnBuf bytes.Buffer

		missing := checkAllowlistCoverage("geosite", allow, optional, kept, &warnBuf)

		if len(missing) != 0 {
			t.Errorf("missing = %v, want none (entry was optional)", missing)
		}
		if !strings.Contains(warnBuf.String(), "geosite:category-ads-ir") {
			t.Errorf("expected a warning naming the optional token, got: %q", warnBuf.String())
		}
	})

	t.Run("everything present -> no missing, no warnings", func(t *testing.T) {
		allow := map[string]bool{"CATEGORY-RU": true}
		optional := map[string]bool{}
		kept := []string{"geosite:category-ru"}
		var warnBuf bytes.Buffer

		missing := checkAllowlistCoverage("geosite", allow, optional, kept, &warnBuf)

		if len(missing) != 0 {
			t.Errorf("missing = %v, want none", missing)
		}
		if warnBuf.Len() != 0 {
			t.Errorf("expected no warnings, got: %q", warnBuf.String())
		}
	})
}
