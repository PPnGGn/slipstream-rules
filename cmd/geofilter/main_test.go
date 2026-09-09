package main

import (
	"os"
	"path/filepath"
	"reflect"
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
		name    string
		body    string
		want    map[string]bool
		keepAll bool
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			set, keepAll, err := readAllowlist(writeTemp(t, tc.body))
			if err != nil {
				t.Fatalf("readAllowlist: %v", err)
			}
			if keepAll != tc.keepAll {
				t.Errorf("keepAll = %v, want %v", keepAll, tc.keepAll)
			}
			if !reflect.DeepEqual(set, tc.want) {
				t.Errorf("set = %v, want %v", set, tc.want)
			}
		})
	}
}

func TestReadAllowlistMissingFile(t *testing.T) {
	if _, _, err := readAllowlist(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
