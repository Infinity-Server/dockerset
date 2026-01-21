package geosite

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// Matcher wraps Xray's DomainMatcher to evaluate routing-style domain rules.
type Matcher struct {
	dm *router.DomainMatcher
}

type yamlConfig struct {
	Domains []string `yaml:"domains"`
}

// LoadMatcherFromYAML loads routing domain rules from a YAML file,
// expanding any geosite references from a provided dat source.
// Supported inputs: ext:file:tag, geosite:tag, domain:, keyword:, full:, dotless:, regexp:, and plain strings (keyword).
func LoadMatcherFromYAML(yamlPath string, dlcPath string) (*Matcher, error) {
	b, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}
	var cfg yamlConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	var domains []*router.Domain
	loader := newRuleLoader(yamlPath, dlcPath)
	for _, item := range cfg.Domains {
		t := strings.TrimSpace(item)
		if t == "" {
			continue
		}
		domains = append(domains, loader.parseItem(t)...)
	}
	m, err := router.NewMphMatcherGroup(domains)
	if err != nil {
		return nil, err
	}
	return &Matcher{dm: m}, nil
}

// ruleLoader centralizes parsing of individual rule items and dat loading.
type ruleLoader struct {
	defaultGeoPath string
	resourceDir    string
	cache          map[string]*router.GeoSiteList
}

func newRuleLoader(yamlPath, dlcPath string) *ruleLoader {
	return &ruleLoader{
		defaultGeoPath: dlcPath,
		resourceDir:    filepath.Dir(yamlPath),
		cache:          make(map[string]*router.GeoSiteList),
	}
}

// getGeo returns a parsed GeoSiteList for the given path, using an in-memory cache.
func (l *ruleLoader) getGeo(p string) *router.GeoSiteList {
	if v, ok := l.cache[p]; ok && v != nil {
		return v
	}
	v := loadGeoSiteList(p)
	if v != nil {
		l.cache[p] = v
	}
	return v
}

// parseItem converts a single rule string to Domain entries.
// ext:file:tag and geosite:tag expand into multiple domains from a dat file.
// Other rule types map to Xray router.Domain semantics.
func (l *ruleLoader) parseItem(s string) []*router.Domain {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "ext:") {
		body := strings.TrimSpace(strings.TrimPrefix(t, "ext:"))
		parts := strings.SplitN(body, ":", 2)
		if len(parts) == 2 {
			file := strings.TrimSpace(parts[0])
			tag := strings.TrimSpace(parts[1])
			var path string
			if filepath.IsAbs(file) {
				path = file
			} else {
				path = filepath.Join(l.resourceDir, file)
			}
			elist := l.getGeo(path)
			return domainsFromGeoSite(elist, tag)
		}
		return nil
	}
	if strings.HasPrefix(t, "geosite:") {
		name := strings.TrimSpace(strings.TrimPrefix(t, "geosite:"))
		elist := l.getGeo(l.defaultGeoPath)
		return domainsFromGeoSite(elist, name)
	}
	return parseDomainItem(t)
}

// loadGeoSiteList reads a protobuf-encoded GeoSiteList from disk.
func loadGeoSiteList(path string) *router.GeoSiteList {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var list router.GeoSiteList
	if err := proto.Unmarshal(b, &list); err != nil {
		return nil
	}
	return &list
}

// parseDomainItem maps a domain selector string to router.Domain entries.
// Plain strings are treated as keyword matches; case-insensitive unless dotless.
func parseDomainItem(s string) []*router.Domain {
	t := strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(t, "regexp:"):
		body := strings.TrimSpace(strings.TrimPrefix(t, "regexp:"))
		return []*router.Domain{{
			Type:  router.Domain_Regex,
			Value: body,
		}}
	case strings.HasPrefix(t, "domain:"):
		body := strings.TrimSpace(strings.TrimPrefix(t, "domain:"))
		return []*router.Domain{{
			Type:  router.Domain_Plain,
			Value: strings.ToLower(body),
		}}
	case strings.HasPrefix(t, "keyword:"):
		body := strings.TrimSpace(strings.TrimPrefix(t, "keyword:"))
		return []*router.Domain{{
			Type:  router.Domain_Plain,
			Value: strings.ToLower(body),
			Attribute: []*router.Domain_Attribute{
				{Key: "keyword", TypedValue: &router.Domain_Attribute_BoolValue{BoolValue: true}},
			},
		}}
	case strings.HasPrefix(t, "full:"):
		body := strings.TrimSpace(strings.TrimPrefix(t, "full:"))
		return []*router.Domain{{
			Type:  router.Domain_Plain,
			Value: strings.ToLower(body),
			Attribute: []*router.Domain_Attribute{
				{Key: "full", TypedValue: &router.Domain_Attribute_BoolValue{BoolValue: true}},
			},
		}}
	case strings.HasPrefix(t, "dotless:"):
		body := strings.TrimSpace(strings.TrimPrefix(t, "dotless:"))
		return []*router.Domain{{
			Type:  router.Domain_Plain,
			Value: body,
			Attribute: []*router.Domain_Attribute{
				{Key: "dotless", TypedValue: &router.Domain_Attribute_BoolValue{BoolValue: true}},
			},
		}}
	default:
		body := strings.ToLower(t)
		return []*router.Domain{{
			Type:  router.Domain_Plain,
			Value: body,
			Attribute: []*router.Domain_Attribute{
				{Key: "keyword", TypedValue: &router.Domain_Attribute_BoolValue{BoolValue: true}},
			},
		}}
	}
}

// domainsFromGeoSite extracts domains by tag (countryCode/name) from a GeoSiteList.
func domainsFromGeoSite(geo *router.GeoSiteList, name string) []*router.Domain {
	if geo == nil {
		return nil
	}
	var out []*router.Domain
	for _, e := range geo.GetEntry() {
		if strings.EqualFold(e.GetCountryCode(), name) {
			for _, d := range e.GetDomain() {
				out = append(out, d)
			}
			break
		}
	}
	return out
}

// Match reports whether the given domain hits the matcher rules.
func (m *Matcher) Match(domain string) bool {
	if m == nil || m.dm == nil {
		return false
	}
	return m.dm.ApplyDomain(strings.TrimSuffix(domain, "."))
}
