package app

import (
	"fmt"
	"net/url"
	"os"

	"geo-sni-proxy/internal/dns"
	"geo-sni-proxy/internal/geosite"
	"geo-sni-proxy/internal/logx"
	"geo-sni-proxy/internal/proxy"
)

// Start bootstraps DNS and HTTP/TLS forwarders using the upstream proxy URI.
func Start() error {
	proxyURI := os.Getenv("GEO_SNI_PROXY_URI")
	if proxyURI == "" {
		fmt.Fprintf(os.Stderr, "GEO_SNI_PROXY_URI is empty, example: socks5://1.2.3.4:5555\n")
		return fmt.Errorf("missing GEO_SNI_PROXY_URI")
	}
	logx.Printf("app_start proxy_uri=%s", proxyURI)
	u, err := url.Parse(proxyURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		fmt.Fprintf(os.Stderr, "Invalid GEO_SNI_PROXY_URI: %s\n", proxyURI)
		return fmt.Errorf("invalid GEO_SNI_PROXY_URI")
	}
	switch u.Scheme {
	case "socks5", "socks5h", "http", "https":
	default:
		fmt.Fprintf(os.Stderr, "Unsupported proxy scheme: %s\n", u.Scheme)
		return fmt.Errorf("unsupported proxy scheme")
	}
	dnsAddr := os.Getenv("GEO_SNI_PROXY_DNS_ADDR")
	if dnsAddr == "" {
		dnsAddr = "127.0.0.1:53"
	}
	httpAddr := os.Getenv("GEO_SNI_PROXY_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = "127.0.0.1:80"
	}
	tlsAddr := os.Getenv("GEO_SNI_PROXY_TLS_ADDR")
	if tlsAddr == "" {
		tlsAddr = "127.0.0.1:443"
	}
	upstreamIP, err := dns.EnsureHostsAndResolv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization failed: %v\n", err)
		return err
	}
	logx.Printf("app_dns_upstream ip=%s", upstreamIP)
	matcher, err := geosite.LoadMatcherFromYAML("/metadata/domains.yaml", "/usr/local/share/xray/geosite.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load matcher: %v\n", err)
		return err
	}
	logx.Printf("app_matcher_loaded source=/metadata/domains.yaml geosite=/usr/local/share/xray/geosite.dat")
	go dns.StartServer(matcher, upstreamIP, dnsAddr)
	_ = proxy.StartHTTPAndTLS(proxyURI, httpAddr, tlsAddr)
	logx.Printf("app_servers_started http=%s tls=%s dns=%s", httpAddr, tlsAddr, dnsAddr)
	return nil
}
