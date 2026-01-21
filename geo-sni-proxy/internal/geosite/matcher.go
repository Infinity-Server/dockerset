package geosite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/xtls/xray-core/app/router"
	xconf "github.com/xtls/xray-core/infra/conf"
	"gopkg.in/yaml.v3"
)

type Matcher struct {
	dm *router.DomainMatcher
}

type yamlConfig struct {
	Domains []string `yaml:"domains"`
}

func LoadMatcherFromYAML(yamlPath string, dlcPath string) (*Matcher, error) {
	b, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}
	_ = os.Setenv("xray.location.asset", filepath.Dir(dlcPath))
	var cfg yamlConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	ruleBody := struct {
		OutboundTag string   `json:"outboundTag"`
		Domains     []string `json:"domains"`
	}{
		OutboundTag: "_",
		Domains:     cfg.Domains,
	}
	jb, err := json.Marshal(ruleBody)
	if err != nil {
		return nil, err
	}
	rule, err := xconf.ParseRule(jb)
	if err != nil {
		return nil, err
	}
	domains := rule.Domain
	mph, err := router.NewMphMatcherGroup(domains)
	if err != nil {
		return nil, err
	}
	return &Matcher{dm: mph}, nil
}

func mapToQuoted(in []string) []string {
	var out []string
	for _, s := range in {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		out = append(out, `"`+t+`"`)
	}
	return out
}

func (m *Matcher) Match(domain string) bool {
	if m == nil {
		return false
	}
	d := strings.ToLower(strings.TrimSuffix(domain, "."))
	if m.dm == nil {
		return false
	}
	return m.dm.ApplyDomain(d)
}
