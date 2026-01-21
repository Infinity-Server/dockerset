package geosite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

func writeGeoSiteDat(path string) error {
	var list router.GeoSiteList
	list.Entry = []*router.GeoSite{
		{
			CountryCode: "TESTTAG",
			Domain: []*router.Domain{
				{Type: router.Domain_Plain, Value: "zoo"},
				{Type: router.Domain_Domain, Value: "github.com"},
				{Type: router.Domain_Full, Value: "api.github.com"},
				{Type: router.Domain_Regex, Value: `^post.*\.github\.io$`},
			},
		},
	}
	b, err := proto.Marshal(&list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func TestMatcherBasic(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("xray.location.asset", tmp)
	dat := filepath.Join(tmp, "test.dat")
	if err := writeGeoSiteDat(dat); err != nil {
		t.Fatalf("failed to write dat: %v", err)
	}

	yaml := `domains:
  - keyword:foo
  - domain:example.com
  - full:exact.example.org
  - regexp:^bar\.example\.net$
  - dotless:abc
  - ext:test.dat:TESTTAG
  - zzz
  - domain:jellyfin.org
`
	ymlPath := filepath.Join(tmp, "rules.yaml")
	if err := os.WriteFile(ymlPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	m, err := LoadMatcherFromYAML(ymlPath, dat)
	if err != nil {
		t.Fatalf("LoadMatcherFromYAML error: %v", err)
	}

	if !m.Match("www.example.com") {
		t.Errorf("domain suffix should match")
	}
	if !m.Match("repo.jellyfin.org") {
		t.Errorf("domain suffix should match")
	}
	if !m.Match("exact.example.org") {
		t.Errorf("full should match")
	}
	if !m.Match("exact.example.org.") {
		t.Errorf("full with trailing dot should match")
	}
	if !m.Match("bar.example.net") {
		t.Errorf("regex should match")
	}
	if m.Match("x.bar.example.net") {
		t.Errorf("regex anchored should not match subdomain")
	}
	if !m.Match("my-foo-domain.com") {
		t.Errorf("keyword should match")
	}
	if !m.Match("abc") || !m.Match("xabcx") {
		t.Errorf("dotless keyword should match non-dot names")
	}
	if m.Match("a.bc") {
		t.Errorf("dotless should not match names with dot")
	}
	if !m.Match("github.com") {
		t.Errorf("geosite domain suffix should match")
	}
	if !m.Match("api.github.com") {
		t.Errorf("geosite full should match")
	}
	if !m.Match("post-1.github.io") {
		t.Errorf("geosite regex should match")
	}
	if !m.Match("azzzc") {
		t.Errorf("plain string keyword should match")
	}
	if m.Match("nomatch.example.xyz") {
		t.Errorf("unexpected match")
	}
}
