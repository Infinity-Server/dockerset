package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"geo-sni-proxy/internal/logx"
)

// StartHTTPAndTLS launches HTTP and TLS transparent forwarders via the upstream proxy.
// httpAddr and tlsAddr specify the local listening addresses, e.g. "127.0.0.1:80".
func StartHTTPAndTLS(proxyURI, httpAddr, tlsAddr string) error {
	go func() { _ = listenHTTP(proxyURI, httpAddr) }()
	go func() { _ = listenTLS(proxyURI, tlsAddr) }()
	return nil
}

func listenHTTP(proxyURI, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	logx.Printf("listen proto=http addr=%s", addr)
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleHTTPConn(c, proxyURI)
	}
}

func listenTLS(proxyURI, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	logx.Printf("listen proto=tls addr=%s", addr)
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleTLSConn(c, proxyURI)
	}
}

// handleHTTPConn parses HTTP headers to extract Host and forwards data through upstream proxy.
func handleHTTPConn(c net.Conn, proxyURI string) {
	defer c.Close()
	remote := c.RemoteAddr().String()
	logx.Printf("accept proto=http remote=%s", remote)
	br := bufio.NewReader(c)
	hdr, host, err := readHTTPHeader(br)
	if err != nil {
		logx.Printf("read_header_error proto=http remote=%s err=%v", remote, err)
		return
	}
	if host == "" {
		logx.Printf("parse_http_host_failed remote=%s", remote)
		return
	}
	pc, err := dialViaProxy(proxyURI, host, 80)
	if err != nil {
		logx.Printf("dial_upstream_error proto=http host=%s via=%s remote=%s err=%v", host, proxyURI, remote, err)
		return
	}
	defer pc.Close()
	logx.Printf("forward_start proto=http host=%s port=80 via=%s remote=%s", host, proxyURI, remote)
	if len(hdr) > 0 {
		_, _ = pc.Write(hdr)
	}
	if br.Buffered() > 0 {
		buf := make([]byte, br.Buffered())
		_, _ = br.Read(buf)
		_, _ = pc.Write(buf)
	}
	pipeAndLog(c, pc, "http", host, 80, remote, proxyURI)
}

// handleTLSConn extracts SNI from ClientHello and forwards data through upstream proxy.
func handleTLSConn(c net.Conn, proxyURI string) {
	defer c.Close()
	remote := c.RemoteAddr().String()
	logx.Printf("accept proto=tls remote=%s", remote)
	br := bufio.NewReader(c)
	hdr := make([]byte, 5)
	if _, err := io.ReadFull(br, hdr); err != nil {
		logx.Printf("tls_read_header_error remote=%s err=%v", remote, err)
		return
	}
	if hdr[0] != 0x16 {
		logx.Printf("tls_not_handshake remote=%s", remote)
		return
	}
	hl := int(binary.BigEndian.Uint16(hdr[3:5]))
	body := make([]byte, hl)
	if _, err := io.ReadFull(br, body); err != nil {
		logx.Printf("tls_read_body_error remote=%s err=%v", remote, err)
		return
	}
	clientHello := append(hdr, body...)
	serverName := parseSNI(clientHello)
	if serverName == "" {
		logx.Printf("parse_sni_failed remote=%s", remote)
		return
	}
	pc, err := dialViaProxy(proxyURI, serverName, 443)
	if err != nil {
		logx.Printf("dial_upstream_error proto=tls sni=%s via=%s remote=%s err=%v", serverName, proxyURI, remote, err)
		return
	}
	defer pc.Close()
	logx.Printf("forward_start proto=tls sni=%s port=443 via=%s remote=%s", serverName, proxyURI, remote)
	if len(clientHello) > 0 {
		_, _ = pc.Write(clientHello)
	}
	if br.Buffered() > 0 {
		buf := make([]byte, br.Buffered())
		_, _ = br.Read(buf)
		_, _ = pc.Write(buf)
	}
	pipeAndLog(c, pc, "tls", serverName, 443, remote, proxyURI)
}

// parseHTTPHost attempts to obtain the Host from headers or absolute-form request line.
func parseHTTPHost(b []byte) string {
	s := string(b)
	lines := strings.Split(s, "\r\n")
	for _, l := range lines {
		lo := strings.ToLower(l)
		if strings.HasPrefix(lo, "host:") {
			h := strings.TrimSpace(strings.TrimPrefix(l, "Host:"))
			h = strings.TrimSpace(strings.TrimPrefix(h, "host:"))
			if h == "" {
				continue
			}
			if j := strings.IndexByte(h, ':'); j > 0 {
				return h[:j]
			}
			return h
		}
	}
	if len(lines) > 0 {
		first := lines[0]
		if strings.Contains(first, " http://") {
			uEnd := strings.Index(first, " HTTP/")
			if uEnd > 0 {
				start := strings.Index(first, "http://")
				u := first[start:uEnd]
				pu, err := url.Parse(u)
				if err == nil && pu.Host != "" {
					if j := strings.IndexByte(pu.Host, ':'); j > 0 {
						return pu.Host[:j]
					}
					return pu.Host
				}
			}
		}
	}
	return ""
}

// readHTTPHeader reads request headers until the blank line and captures Host.
func readHTTPHeader(br *bufio.Reader) ([]byte, string, error) {
	var buf bytes.Buffer
	var host string
	for {
		l, err := br.ReadString('\n')
		if err != nil {
			return nil, "", err
		}
		buf.WriteString(l)
		lo := strings.ToLower(l)
		if strings.HasPrefix(lo, "host:") && host == "" {
			h := strings.TrimSpace(strings.TrimPrefix(l, "Host:"))
			h = strings.TrimSpace(strings.TrimPrefix(h, "host:"))
			if j := strings.IndexByte(h, ':'); j > 0 {
				host = h[:j]
			} else {
				host = h
			}
		}
		if l == "\r\n" {
			break
		}
	}
	if host == "" {
		host = parseHTTPHost(buf.Bytes())
	}
	return buf.Bytes(), host, nil
}

// parseSNI extracts the server_name from a TLS ClientHello.
func parseSNI(b []byte) string {
	if len(b) < 5 || b[0] != 0x16 {
		return ""
	}
	handshakeLen := int(binary.BigEndian.Uint16(b[3:5]))
	if len(b) < 5+handshakeLen {
		return ""
	}
	offset := 5
	if b[offset] != 0x01 {
		return ""
	}
	offset++
	if offset+3 > len(b) {
		return ""
	}
	_ = int(b[offset])<<16 | int(b[offset+1])<<8 | int(b[offset+2])
	offset += 3
	offset += 2 // version
	offset += 32
	if offset >= len(b) {
		return ""
	}
	if offset+1 > len(b) {
		return ""
	}
	sessionIDLen := int(b[offset])
	offset += 1 + sessionIDLen
	if offset+2 > len(b) {
		return ""
	}
	cipherSuiteLen := int(binary.BigEndian.Uint16(b[offset : offset+2]))
	offset += 2 + cipherSuiteLen
	if offset+1 > len(b) {
		return ""
	}
	compLen := int(b[offset])
	offset += 1 + compLen
	if offset+2 > len(b) {
		return ""
	}
	extLen := int(binary.BigEndian.Uint16(b[offset : offset+2]))
	offset += 2
	end := offset + extLen
	for offset+4 <= end && end <= len(b) {
		typ := int(binary.BigEndian.Uint16(b[offset : offset+2]))
		l := int(binary.BigEndian.Uint16(b[offset+2 : offset+4]))
		offset += 4
		if typ == 0x00 { // server_name
			if offset+2 > len(b) {
				return ""
			}
			listLen := int(binary.BigEndian.Uint16(b[offset : offset+2]))
			offset += 2
			if offset+listLen > len(b) {
				return ""
			}
			cur := offset
			for cur+3 <= offset+listLen {
				nameType := int(b[cur])
				nameLen := int(binary.BigEndian.Uint16(b[cur+1 : cur+3]))
				cur += 3
				if nameType == 0 {
					if cur+nameLen > len(b) {
						return ""
					}
					return strings.ToLower(string(b[cur : cur+nameLen]))
				}
				cur += nameLen
			}
			return ""
		}
		offset += l
	}
	return ""
}

// dialViaProxy establishes a TCP tunnel via the configured upstream proxy.
func dialViaProxy(proxyURI, host string, port int) (net.Conn, error) {
	u, err := url.Parse(proxyURI)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "socks5", "socks5h":
		logx.Printf("dial_scheme=socks5 upstream=%s target=%s:%d", u.Host, host, port)
		return dialSOCKS5(u, host, port)
	case "http":
		logx.Printf("dial_scheme=http upstream=%s target=%s:%d", u.Host, host, port)
		return dialHTTPProxy(u, host, port, false)
	case "https":
		logx.Printf("dial_scheme=https upstream=%s target=%s:%d", u.Host, host, port)
		return dialHTTPProxy(u, host, port, true)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

// dialSOCKS5 performs a SOCKS5 CONNECT to host:port without authentication.
func dialSOCKS5(u *url.URL, host string, port int) (net.Conn, error) {
	addr := u.Host
	c, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	// no-auth
	_, err = c.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		c.Close()
		return nil, err
	}
	rep := make([]byte, 2)
	if _, err = io.ReadFull(c, rep); err != nil {
		c.Close()
		return nil, err
	}
	hostb := []byte(host)
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(0x05)
	buf.WriteByte(0x01)
	buf.WriteByte(0x00)
	buf.WriteByte(0x03)
	buf.WriteByte(byte(len(hostb)))
	buf.Write(hostb)
	var p [2]byte
	binary.BigEndian.PutUint16(p[:], uint16(port))
	buf.Write(p[:])
	if _, err = c.Write(buf.Bytes()); err != nil {
		c.Close()
		return nil, err
	}
	hdr := make([]byte, 4)
	if _, err = io.ReadFull(c, hdr); err != nil {
		c.Close()
		return nil, err
	}
	var addrLen int
	switch hdr[3] {
	case 0x01:
		addrLen = 4
	case 0x03:
		lb := make([]byte, 1)
		if _, err = io.ReadFull(c, lb); err != nil {
			c.Close()
			return nil, err
		}
		addrLen = int(lb[0])
	case 0x04:
		addrLen = 16
	}
	if addrLen > 0 {
		tmp := make([]byte, addrLen+2)
		if _, err = io.ReadFull(c, tmp); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, nil
}

// dialHTTPProxy performs an HTTP/HTTPS CONNECT to host:port through the proxy.
func dialHTTPProxy(u *url.URL, host string, port int, tlsProxy bool) (net.Conn, error) {
	addr := u.Host
	var c net.Conn
	var err error
	if tlsProxy {
		tc, err2 := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{InsecureSkipVerify: true, ServerName: u.Hostname()})
		if err2 != nil {
			return nil, err2
		}
		c = tc
	} else {
		c, err = net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return nil, err
		}
	}
	target := fmt.Sprintf("%s:%d", host, port)
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: Keep-Alive\r\n\r\n", target, target)
	if _, err = c.Write([]byte(req)); err != nil {
		c.Close()
		return nil, err
	}
	br := bufio.NewReader(c)
	line, err := br.ReadString('\n')
	if err != nil {
		c.Close()
		return nil, err
	}
	if !strings.Contains(line, "200") {
		c.Close()
		return nil, fmt.Errorf("proxy connect failed: %s", strings.TrimSpace(line))
	}
	logx.Printf("proxy_connect_ok upstream=%s target=%s:%d status_line=%q", u.Host, host, port, strings.TrimSpace(line))
	// drain headers
	for {
		l, err := br.ReadString('\n')
		if err != nil {
			c.Close()
			return nil, err
		}
		if l == "\r\n" {
			break
		}
	}
	return c, nil
}

// pipeAndLog bidirectionally copies data between client and upstream and logs transfer stats.
func pipeAndLog(client, upstream net.Conn, proto, host string, port int, remote, via string) {
	var up, down int64
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		n, err := io.Copy(upstream, client)
		atomic.AddInt64(&up, n)
		if err != nil {
			logx.Printf("copy_error dir=client_to_upstream proto=%s host=%s remote=%s err=%v", proto, host, remote, err)
		}
		closeWrite(upstream)
		wg.Done()
	}()
	go func() {
		n, err := io.Copy(client, upstream)
		atomic.AddInt64(&down, n)
		if err != nil {
			logx.Printf("copy_error dir=upstream_to_client proto=%s host=%s remote=%s err=%v", proto, host, remote, err)
		}
		closeWrite(client)
		wg.Done()
	}()
	wg.Wait()
	dur := time.Since(start)
	logx.Printf("forward_done proto=%s host=%s port=%d via=%s remote=%s bytes_up=%d bytes_down=%d duration_ms=%d", proto, host, port, via, remote, up, down, dur.Milliseconds())
	client.Close()
	upstream.Close()
}

// closeWrite half-closes TCP connections where supported.
func closeWrite(conn net.Conn) {
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
		return
	}
	_ = conn.Close()
}
