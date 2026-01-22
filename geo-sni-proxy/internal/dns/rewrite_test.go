package dns

import (
	"net"
	"testing"

	mdns "github.com/miekg/dns"
)

func TestRewriteAnswersCNAMEToA(t *testing.T) {
	ipA := net.IPv4(10, 0, 0, 2)
	match := func(s string) bool { return s == "example.com" }
	in := []mdns.RR{
		&mdns.CNAME{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeCNAME, Class: mdns.ClassINET, Ttl: 60}, Target: "target.example.com."},
	}
	out, changed := rewriteAnswers(match, in, ipA)
	if !changed {
		t.Fatalf("expected changed")
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 rr, got %d", len(out))
	}
	if a, ok := out[0].(*mdns.A); !ok || !a.A.Equal(ipA) || a.Hdr.Name != "example.com." {
		t.Fatalf("unexpected rr: %#v", out[0])
	}
}

func TestRewriteAnswersDropAAAA(t *testing.T) {
	ipA := net.IPv4(10, 0, 0, 2)
	match := func(s string) bool { return s == "example.com" }
	in := []mdns.RR{
		&mdns.AAAA{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeAAAA, Class: mdns.ClassINET, Ttl: 60}, AAAA: net.IPv6loopback},
	}
	out, changed := rewriteAnswers(match, in, ipA)
	if !changed {
		t.Fatalf("expected changed")
	}
	if len(out) != 0 {
		t.Fatalf("expected 0 rr, got %d", len(out))
	}
}

