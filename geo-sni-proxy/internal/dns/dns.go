package dns

import (
	"net"
	"os"
	"strings"
	"time"

	"geo-sni-proxy/internal/geosite"
	"geo-sni-proxy/internal/logx"

	mdns "github.com/miekg/dns"
)

// EnsureHostsAndResolv configures /etc/hosts and /etc/resolv.conf
// so that the local DNS server becomes the primary nameserver.
// It returns the detected upstream resolver IP.
func EnsureHostsAndResolv() (string, error) {
	const domain = "upstream-dns.geo-sni-proxy"
	hosts := "/etc/hosts"
	resolv := "/etc/resolv.conf"
	hb, err := os.ReadFile(hosts)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(hb), "\n")
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		fields := strings.Fields(t)
		if len(fields) >= 2 {
			for _, name := range fields[1:] {
				if name == domain {
					return fields[0], nil
				}
			}
		}
	}
	rb, err := os.ReadFile(resolv)
	if err != nil {
		return "", err
	}
	var upstream string
	rLines := strings.Split(string(rb), "\n")
	for _, l := range rLines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasPrefix(t, "nameserver") {
			fs := strings.Fields(t)
			if len(fs) >= 2 {
				ip := fs[1]
				if ip != "127.0.0.1" && ip != "0.0.0.0" {
					upstream = ip
					break
				}
			}
		}
	}
	if upstream == "" {
		upstream = "8.8.8.8"
	}
	appendLine := upstream + " " + domain + "\n"
	f, err := os.OpenFile(hosts, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(appendLine); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	logx.Printf("dns_upstream_resolved ip=%s hosts_appended_domain=%s", upstream, domain)
	var out []string
	for _, l := range rLines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "nameserver") {
			out = append(out, "nameserver 127.0.0.1")
		} else if t != "" {
			out = append(out, l)
		} else {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		out = append(out, "nameserver 127.0.0.1")
	}
	content := strings.Join(out, "\n")
	if !strings.Contains(content, "nameserver 127.0.0.1") {
		content = "nameserver 127.0.0.1\n" + content
	}
	if err := os.WriteFile(resolv, []byte(content), 0644); err != nil {
		return "", err
	}
	logx.Printf("dns_resolv_updated nameserver=127.0.0.1")
	return upstream, nil
}

// StartServer launches UDP and TCP DNS servers on the provided address.
// Queries hitting the matcher are answered locally; others are proxied upstream.
func StartServer(m *geosite.Matcher, upstream string, addr string) {
	udp := &mdns.Server{Addr: addr, Net: "udp"}
	tcp := &mdns.Server{Addr: addr, Net: "tcp"}
	var ipA = net.IPv4(127, 0, 0, 1)
	if envEnabled("GEO_SNI_PROXY_BYPASS") {
		if bp := findBypassIP(); bp != nil {
			if v4 := bp.To4(); v4 != nil {
				ipA = v4
			}
			logx.Printf("dns_bypass_enabled ip=%s", ipA.String())
		} else {
			logx.Printf("dns_bypass_enabled ip=none")
		}
	}
	logx.Printf("dns_listen start udp=%s tcp=%s upstream=%s", addr, addr, upstream)
	mdns.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		up := queryUpstreamMsg(upstream, r)
		resp := new(mdns.Msg)
		resp.SetReply(r)
		if up != nil {
			resp.MsgHdr.Rcode = up.MsgHdr.Rcode
			resp.MsgHdr.RecursionAvailable = up.MsgHdr.RecursionAvailable
			resp.MsgHdr.Authoritative = up.MsgHdr.Authoritative
			resp.MsgHdr.Truncated = up.MsgHdr.Truncated
			resp.Answer = append(resp.Answer, up.Answer...)
			resp.Ns = append(resp.Ns, up.Ns...)
			resp.Extra = append(resp.Extra, up.Extra...)
		}
		out, changed := rewriteAnswers(func(s string) bool { return m.Match(s) }, resp.Answer, ipA)
		if changed {
			resp.Answer = out
			resp.MsgHdr.Rcode = mdns.RcodeSuccess
		}
		_ = w.WriteMsg(resp)
	})
	go func() { _ = udp.ListenAndServe() }()
	go func() { _ = tcp.ListenAndServe() }()
	select {}
}

func rewriteAnswers(match func(string) bool, answers []mdns.RR, ipA net.IP) ([]mdns.RR, bool) {
	changed := false
	var out []mdns.RR
	for _, rr := range answers {
		h := rr.Header()
		name := strings.TrimSuffix(h.Name, ".")
		if match(name) {
			switch h.Rrtype {
			case mdns.TypeA:
				changed = true
				out = append(out, &mdns.A{
					Hdr: mdns.RR_Header{Name: h.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: h.Ttl},
					A:   ipA,
				})
				continue
			case mdns.TypeAAAA:
				changed = true
				continue
			case mdns.TypeCNAME:
				changed = true
				out = append(out, &mdns.A{
					Hdr: mdns.RR_Header{Name: h.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: h.Ttl},
					A:   ipA,
				})
				continue
			case mdns.TypeHTTPS, mdns.TypeSVCB:
				changed = true
				continue
			}
		}
		out = append(out, rr)
	}
	return out, changed
}

func envEnabled(name string) bool {
	s := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch s {
	case "1", "true", "yes", "on", "enable", "enabled":
		return true
	default:
		return false
	}
}

func findBypassIP() net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, inf := range ifaces {
		if (inf.Flags&net.FlagUp) == 0 || (inf.Flags&net.FlagLoopback) != 0 {
			continue
		}
		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}
		if ip := chooseBypassIPv4FromAddrs(addrs); ip != nil {
			return ip
		}
	}
	return nil
}

func chooseBypassIPv4FromAddrs(addrs []net.Addr) net.IP {
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok {
			ip := ipnet.IP
			if ip == nil {
				continue
			}
			v4 := ip.To4()
			if v4 == nil {
				continue
			}
			if v4.IsLoopback() || v4.IsUnspecified() || v4.IsLinkLocalUnicast() {
				continue
			}
			return v4
		}
	}
	return nil
}

// queryUpstream forwards a single question to the upstream resolver.
// It retries over TCP when UDP returns truncated responses.
func queryUpstream(upstream string, q mdns.Question, r *mdns.Msg) *mdns.Msg {
	msg := new(mdns.Msg)
	msg.RecursionDesired = r.RecursionDesired
	msg.Question = []mdns.Question{q}
	c := &mdns.Client{Net: "udp", Timeout: 3 * time.Second}
	addr := net.JoinHostPort(upstream, "53")
	resp, _, err := c.Exchange(msg, addr)
	if err != nil || (resp != nil && resp.Truncated) {
		c = &mdns.Client{Net: "tcp", Timeout: 4 * time.Second}
		resp, _, err = c.Exchange(msg, addr)
		if err != nil {
			logx.Printf("dns_upstream_error name=%s type=%d upstream=%s err=%v", q.Name, q.Qtype, upstream, err)
			return nil
		}
	}
	if resp == nil {
		return nil
	}
	logx.Printf("dns_upstream_answer name=%s type=%d count=%d", q.Name, q.Qtype, len(resp.Answer))
	return resp
}

// queryUpstreamMsg forwards the entire inbound DNS message's questions upstream in one exchange.
func queryUpstreamMsg(upstream string, r *mdns.Msg) *mdns.Msg {
	msg := new(mdns.Msg)
	msg.RecursionDesired = r.RecursionDesired
	msg.Question = append(msg.Question, r.Question...)
	c := &mdns.Client{Net: "udp", Timeout: 3 * time.Second}
	addr := net.JoinHostPort(upstream, "53")
	resp, _, err := c.Exchange(msg, addr)
	if err != nil || (resp != nil && resp.Truncated) {
		c = &mdns.Client{Net: "tcp", Timeout: 4 * time.Second}
		resp, _, err = c.Exchange(msg, addr)
		if err != nil {
			logx.Printf("dns_upstream_error_batch count=%d upstream=%s err=%v", len(r.Question), upstream, err)
			return nil
		}
	}
	if resp == nil {
		return nil
	}
	logx.Printf("dns_upstream_answer_batch qcount=%d ans_count=%d", len(r.Question), len(resp.Answer))
	return resp
}

// filterAnswers removes answers that match the given name and any of the provided types.
func filterAnswers(ans []mdns.RR, name string, types []uint16) []mdns.RR {
	if len(ans) == 0 {
		return ans
	}
	var out []mdns.RR
	for _, rr := range ans {
		h := rr.Header()
		if strings.EqualFold(h.Name, name) {
			matchType := false
			for _, t := range types {
				if h.Rrtype == t {
					matchType = true
					break
				}
			}
			if matchType {
				continue
			}
		}
		out = append(out, rr)
	}
	return out
}

// filterAnswersByName removes all answers for the given name regardless of type.
func filterAnswersByName(ans []mdns.RR, name string) []mdns.RR {
	if len(ans) == 0 {
		return ans
	}
	var out []mdns.RR
	for _, rr := range ans {
		if !strings.EqualFold(rr.Header().Name, name) {
			out = append(out, rr)
		}
	}
	return out
}
