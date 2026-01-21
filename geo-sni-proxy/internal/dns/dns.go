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
		changed := false
		var out []mdns.RR
		for _, rr := range resp.Answer {
			h := rr.Header()
			name := strings.TrimSuffix(h.Name, ".")
			if m.Match(name) {
				switch h.Rrtype {
				case mdns.TypeA:
					changed = true
					out = append(out, &mdns.A{
						Hdr: mdns.RR_Header{Name: h.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: h.Ttl},
						A:   net.IPv4(127, 0, 0, 1),
					})
					continue
				case mdns.TypeAAAA:
					changed = true
					// drop AAAA
					continue
				case mdns.TypeCNAME:
					changed = true
					out = append(out, &mdns.CNAME{
						Hdr:    mdns.RR_Header{Name: h.Name, Rrtype: mdns.TypeCNAME, Class: mdns.ClassINET, Ttl: h.Ttl},
						Target: "localhost.",
					})
					hasLocalhostA := false
					for _, ex := range resp.Extra {
						if ah := ex.Header(); ah.Rrtype == mdns.TypeA && strings.EqualFold(ah.Name, "localhost.") {
							hasLocalhostA = true
							break
						}
					}
					if !hasLocalhostA {
						resp.Extra = append(resp.Extra, &mdns.A{
							Hdr: mdns.RR_Header{Name: "localhost.", Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
							A:   net.IPv4(127, 0, 0, 1),
						})
					}
					continue
				case mdns.TypeHTTPS, mdns.TypeSVCB:
					changed = true
					// drop HTTPS/SVCB
					continue
				}
			}
			out = append(out, rr)
		}
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
