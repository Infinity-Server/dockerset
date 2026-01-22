package dns

import (
	"net"
	"testing"
)

func TestChooseBypassIPv4FromAddrs(t *testing.T) {
	var addrs []net.Addr
	addrs = append(addrs, &net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)})
	addrs = append(addrs, &net.IPNet{IP: net.IPv6loopback, Mask: net.CIDRMask(128, 128)})
	addrs = append(addrs, &net.IPNet{IP: net.IPv4(192, 168, 1, 10), Mask: net.IPv4Mask(255, 255, 255, 0)})
	ip := chooseBypassIPv4FromAddrs(addrs)
	if ip == nil || ip.String() != "192.168.1.10" {
		t.Fatalf("unexpected ip: %v", ip)
	}
}

func TestChooseBypassIPv4FromAddrsNone(t *testing.T) {
	var addrs []net.Addr
	addrs = append(addrs, &net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)})
	ip := chooseBypassIPv4FromAddrs(addrs)
	if ip != nil {
		t.Fatalf("expected nil, got: %v", ip)
	}
}
