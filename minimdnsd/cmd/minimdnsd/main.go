package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	mdnsPort      = 5353
	maxPacketSize = 9036 // RFC 6762 section 6.1 upper bound
	maxNameLength = 255
)

var (
	group4 = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: mdnsPort}
	group6 = &net.UDPAddr{IP: net.ParseIP("ff02::fb"), Port: mdnsPort}
)

type dnsHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type dnsQuestion struct {
	Name   string
	QType  uint16
	QClass uint16
}

type queryIntent struct {
	WantA    bool
	WantAAAA bool
}

func normalizeHostname(v string) (string, error) {
	name := strings.TrimSpace(strings.ToLower(v))
	if len(name) == 0 {
		return "", errors.New("hostname is empty")
	}
	if len(name) > 63 {
		return "", fmt.Errorf("hostname label too long: %d > 63", len(name))
	}
	for _, r := range name {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum || r == '-' {
			continue
		}
		return "", fmt.Errorf("hostname contains invalid char %q", r)
	}
	return name, nil
}

func readHostnameFromFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	return normalizeHostname(line)
}

func readHeader(packet []byte) (dnsHeader, error) {
	if len(packet) < 12 {
		return dnsHeader{}, errors.New("packet too short")
	}
	return dnsHeader{
		ID:      binary.BigEndian.Uint16(packet[0:2]),
		Flags:   binary.BigEndian.Uint16(packet[2:4]),
		QDCount: binary.BigEndian.Uint16(packet[4:6]),
		ANCount: binary.BigEndian.Uint16(packet[6:8]),
		NSCount: binary.BigEndian.Uint16(packet[8:10]),
		ARCount: binary.BigEndian.Uint16(packet[10:12]),
	}, nil
}

func parseName(packet []byte, off *int) (string, error) {
	labels, next, err := parseLabels(packet, *off, 0)
	if err != nil {
		return "", err
	}
	*off = next
	name := strings.Join(labels, ".")
	if len(name) > maxNameLength {
		return "", errors.New("dns name too long")
	}
	return strings.ToLower(name), nil
}

func parseLabels(packet []byte, start int, depth int) ([]string, int, error) {
	if depth > 10 {
		return nil, 0, errors.New("too many compression pointers")
	}
	if start < 0 || start >= len(packet) {
		return nil, 0, errors.New("name offset out of range")
	}

	var labels []string
	i := start
	for {
		if i >= len(packet) {
			return nil, 0, errors.New("truncated name")
		}
		l := packet[i]
		switch l & 0xC0 {
		case 0x00:
			if l == 0 {
				return labels, i + 1, nil
			}
			if l > 63 {
				return nil, 0, errors.New("invalid label length")
			}
			i++
			end := i + int(l)
			if end > len(packet) {
				return nil, 0, errors.New("label out of bounds")
			}
			labels = append(labels, string(packet[i:end]))
			i = end
		case 0xC0:
			if i+1 >= len(packet) {
				return nil, 0, errors.New("truncated compression pointer")
			}
			ptr := int(l&0x3F)<<8 | int(packet[i+1])
			ptrLabels, _, err := parseLabels(packet, ptr, depth+1)
			if err != nil {
				return nil, 0, err
			}
			labels = append(labels, ptrLabels...)
			return labels, i + 2, nil
		default:
			return nil, 0, errors.New("invalid label encoding")
		}
	}
}

func parseQuestions(packet []byte, qdCount uint16) ([]dnsQuestion, error) {
	off := 12
	questions := make([]dnsQuestion, 0, qdCount)
	for i := 0; i < int(qdCount); i++ {
		name, err := parseName(packet, &off)
		if err != nil {
			return nil, err
		}
		if off+4 > len(packet) {
			return nil, errors.New("truncated question")
		}
		questions = append(questions, dnsQuestion{
			Name:   name,
			QType:  binary.BigEndian.Uint16(packet[off : off+2]),
			QClass: binary.BigEndian.Uint16(packet[off+2 : off+4]),
		})
		off += 4
	}
	return questions, nil
}

func parseIntent(packet []byte, hostnameFQDN string) (dnsHeader, queryIntent, error) {
	hdr, err := readHeader(packet)
	if err != nil {
		return dnsHeader{}, queryIntent{}, err
	}

	// Ignore responses.
	if hdr.Flags&0x8000 != 0 {
		return hdr, queryIntent{}, nil
	}

	questions, err := parseQuestions(packet, hdr.QDCount)
	if err != nil {
		return dnsHeader{}, queryIntent{}, err
	}

	intent := queryIntent{}
	for _, q := range questions {
		nameMatches := q.Name == hostnameFQDN
		if !nameMatches {
			continue
		}
		switch q.QType {
		case 1:
			intent.WantA = true
		case 28:
			intent.WantAAAA = true
		case 255:
			intent.WantA = true
			intent.WantAAAA = true
		}
	}
	return hdr, intent, nil
}

func encodeName(name string) ([]byte, error) {
	parts := strings.Split(name, ".")
	out := make([]byte, 0, len(name)+2)
	for _, p := range parts {
		if len(p) == 0 || len(p) > 63 {
			return nil, fmt.Errorf("invalid label length in %q", p)
		}
		out = append(out, byte(len(p)))
		out = append(out, p...)
	}
	out = append(out, 0)
	if len(out) > maxNameLength+1 {
		return nil, errors.New("encoded name too long")
	}
	return out, nil
}

func ifaceUsableIPs(iface *net.Interface, wantV6 bool) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	out := make([]net.IP, 0, 2)
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
			continue
		}
		if wantV6 {
			if ip.To4() != nil || !isUsableIPv6(ip) {
				continue
			}
		} else {
			v4 := ip.To4()
			if v4 == nil || v4[0] == 127 {
				continue
			}
			ip = v4
		}
		out = append(out, ip)
		if len(out) >= 3 {
			return out
		}
	}
	return out
}

func sourceIPMatchesInterface(iface *net.Interface, srcIP net.IP, ipv6 bool) bool {
	if srcIP == nil {
		return false
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet == nil {
			continue
		}
		if ipv6 {
			if srcIP.To4() != nil {
				continue
			}
			// Skip non-IPv6 iface addresses
			if ipNet.IP.To4() != nil || ipNet.IP.To16() == nil {
				continue
			}
			if ipNet.Contains(srcIP) {
				return true
			}
			continue
		}
		src4 := srcIP.To4()
		if src4 == nil {
			continue
		}
		if ipNet.IP.To4() == nil {
			continue
		}
		if ipNet.Contains(src4) {
			return true
		}
	}
	return false
}

func isUsableIPv6(ip net.IP) bool {
	if ip.To16() == nil || ip.To4() != nil {
		return false
	}
	// Exclude loopback and link-local by default; many clients cannot use them reliably.
	return !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

func buildResponse(id uint16, name string, addrs []net.IP, rrType uint16) ([]byte, error) {
	if len(addrs) == 0 {
		return nil, nil
	}
	encName, err := encodeName(name)
	if err != nil {
		return nil, err
	}

	answers := make([]net.IP, 0, len(addrs))
	for _, ip := range addrs {
		if rrType == 1 && ip.To4() != nil {
			answers = append(answers, ip.To4())
		}
		if rrType == 28 && ip.To4() == nil && ip.To16() != nil {
			answers = append(answers, ip.To16())
		}
	}
	if len(answers) == 0 {
		return nil, nil
	}

	packet := make([]byte, 12, 512)
	binary.BigEndian.PutUint16(packet[0:2], id)
	binary.BigEndian.PutUint16(packet[2:4], 0x8400) // response + authoritative
	binary.BigEndian.PutUint16(packet[4:6], 0)      // no question in reply
	binary.BigEndian.PutUint16(packet[6:8], uint16(len(answers)))
	binary.BigEndian.PutUint16(packet[8:10], 0)
	binary.BigEndian.PutUint16(packet[10:12], 0)

	for _, ip := range answers {
		packet = append(packet, encName...)
		packet = append(packet, byte(rrType>>8), byte(rrType))
		packet = append(packet, 0x80, 0x01) // cache flush + IN class
		packet = append(packet, 0x00, 0x00, 0x00, 0xF0)
		if rrType == 1 {
			packet = append(packet, 0x00, 0x04)
			packet = append(packet, ip.To4()...)
		} else {
			packet = append(packet, 0x00, 0x10)
			packet = append(packet, ip.To16()...)
		}
		if len(packet) > maxPacketSize {
			return nil, errors.New("response too large")
		}
	}
	return packet, nil
}

func serveUDP(conn *net.UDPConn, iface *net.Interface, group *net.UDPAddr, hostname string, ipv6 bool) {
	buf := make([]byte, maxPacketSize)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			log.Printf("read error (%v): %v", conn.LocalAddr(), err)
			continue
		}
		if n <= 0 {
			continue
		}
		if !sourceIPMatchesInterface(iface, src.IP, ipv6) {
			// In multi-homed pods, duplicate multicast delivery can happen across listeners.
			// Only answer from the interface whose subnet contains the query source.
			continue
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		hdr, intent, err := parseIntent(packet, hostname+".local")
		if err != nil {
			continue
		}

		if intent.WantA && !ipv6 {
			resp, err := buildResponse(hdr.ID, hostname+".local", ifaceUsableIPs(iface, false), 1)
			if err == nil && len(resp) > 0 {
				_, _ = conn.WriteToUDP(resp, src)
				_, _ = conn.WriteToUDP(resp, group)
			}
		}
		if intent.WantAAAA && ipv6 {
			resp, err := buildResponse(hdr.ID, hostname+".local", ifaceUsableIPs(iface, true), 28)
			if err == nil && len(resp) > 0 {
				_, _ = conn.WriteToUDP(resp, src)
				_, _ = conn.WriteToUDP(resp, group)
			}
		}
	}
}

func parseInterfaceAllowList(v string) map[string]struct{} {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	allow := map[string]struct{}{}
	for _, part := range strings.Split(v, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		allow[name] = struct{}{}
	}
	if len(allow) == 0 {
		return nil
	}
	return allow
}

func selectInterfaces(allow map[string]struct{}) ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	selected := make([]net.Interface, 0, len(ifaces))
	seenAllowed := map[string]bool{}
	for _, iface := range ifaces {
		if allow != nil {
			if _, ok := allow[iface.Name]; !ok {
				continue
			}
			seenAllowed[iface.Name] = true
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		selected = append(selected, iface)
	}
	if allow != nil {
		for name := range allow {
			if !seenAllowed[name] {
				return nil, fmt.Errorf("interface not found: %s", name)
			}
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no usable multicast interfaces")
	}
	return selected, nil
}

func setMulticastInterface(conn *net.UDPConn, iface *net.Interface, ipv6 bool) error {
	rc, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var serr error
	ctrlErr := rc.Control(func(fd uintptr) {
		if ipv6 {
			serr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_IF, iface.Index)
			return
		}
		v4 := ifaceUsableIPs(iface, false)
		if len(v4) == 0 {
			serr = fmt.Errorf("interface %s has no usable IPv4", iface.Name)
			return
		}
		ip4 := v4[0].To4()
		if ip4 == nil {
			serr = fmt.Errorf("interface %s has invalid IPv4", iface.Name)
			return
		}
		var addr [4]byte
		copy(addr[:], ip4)
		serr = syscall.SetsockoptInet4Addr(int(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, addr)
	})
	if ctrlErr != nil {
		return ctrlErr
	}
	return serr
}

func run(hostname string, ipv4Only bool, ifaceSelector string) error {
	allow := parseInterfaceAllowList(ifaceSelector)
	ifaces, err := selectInterfaces(allow)
	if err != nil {
		return err
	}

	listenerCount := 0
	for _, iface := range ifaces {
		iface := iface
		conn4, err := net.ListenMulticastUDP("udp4", &iface, group4)
		if err != nil {
			log.Printf("warning: ipv4 listen failed on %s: %v", iface.Name, err)
		} else {
			if err := conn4.SetReadBuffer(maxPacketSize * 4); err != nil {
				log.Printf("warning: cannot set read buffer for ipv4 (%s): %v", iface.Name, err)
			}
			if err := setMulticastInterface(conn4, &iface, false); err != nil {
				log.Printf("warning: cannot set ipv4 multicast iface %s: %v", iface.Name, err)
			}
			log.Printf("listening ipv4 on %s (%s) for %s.local", group4.String(), iface.Name, hostname)
			go serveUDP(conn4, &iface, group4, hostname, false)
			listenerCount++
		}

		if ipv4Only {
			continue
		}
		conn6, err := net.ListenMulticastUDP("udp6", &iface, group6)
		if err != nil {
			log.Printf("warning: ipv6 listen failed on %s: %v", iface.Name, err)
			continue
		}
		if err := conn6.SetReadBuffer(maxPacketSize * 4); err != nil {
			log.Printf("warning: cannot set read buffer for ipv6 (%s): %v", iface.Name, err)
		}
		if err := setMulticastInterface(conn6, &iface, true); err != nil {
			log.Printf("warning: cannot set ipv6 multicast iface %s: %v", iface.Name, err)
		}
		log.Printf("listening ipv6 on [%s]:%d (%s) for %s.local", group6.IP.String(), group6.Port, iface.Name, hostname)
		go serveUDP(conn6, &iface, group6, hostname, true)
		listenerCount++
	}

	if listenerCount == 0 {
		return errors.New("no listeners started")
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	return nil
}

func main() {
	hostOverride := flag.String("h", "", "override hostname")
	ipv4Only := flag.Bool("4", false, "disable ipv6 listener")
	ifaceSelector := flag.String("i", "", "interface name or comma-separated names (e.g. eth0 or eth0,net1)")
	flag.Parse()

	var hostname string
	var err error
	if *hostOverride != "" {
		hostname, err = normalizeHostname(*hostOverride)
	} else {
		hostname, err = readHostnameFromFile("/etc/hostname")
	}
	if err != nil {
		log.Fatalf("cannot load hostname: %v", err)
	}

	// Delay startup log flush for short-lived supervision restarts.
	time.Sleep(10 * time.Millisecond)
	log.Printf("responding to %s.local", hostname)

	if err := run(hostname, *ipv4Only, *ifaceSelector); err != nil {
		log.Fatal(err)
	}
}
