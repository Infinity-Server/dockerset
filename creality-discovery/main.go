package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type Config struct {
	TargetService string
	ProxyPorts    []string      // Format: "4408,8000"
	ScanInterval  time.Duration
}

var (
	currentIP string
	mu        sync.RWMutex
	cfg       Config
)

func init() {
	cfg = Config{
		TargetService: getEnv("TARGET_SERVICE", "_Creality-1234567890ABCD._udp"),
		ProxyPorts:    strings.Split(getEnv("PROXY_PORTS", "4408,8000"), ","),
		ScanInterval:  time.Duration(getEnvInt("SCAN_INTERVAL_SEC", 30)) * time.Second,
	}
}

func main() {
	// Start mDNS discovery
	go scannerLoop()

	// Start a TCP listener for every configured port
	for _, port := range cfg.ProxyPorts {
		p := strings.TrimSpace(port)
		if p == "" {
			continue
		}
		go startProxy(p)
	}

	// Keep main alive
	select {}
}

func scannerLoop() {
	log.Printf("Discovery started for service: %s", cfg.TargetService)
	for {
		scan()
		time.Sleep(cfg.ScanInterval)
	}
}

func scan() {
	ifaces, _ := net.Interfaces()
	entriesCh := make(chan *mdns.ServiceEntry, 10)

	go func() {
		for entry := range entriesCh {
			if strings.Contains(entry.Name, cfg.TargetService) && entry.AddrV4 != nil {
				ip := entry.AddrV4.String()
				if ip == "0.0.0.0" {
					continue
				}
				mu.Lock()
				if currentIP != ip {
					currentIP = ip
					log.Printf("Target IP Updated: %s", currentIP)
				}
				mu.Unlock()
			}
		}
	}()

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		params := &mdns.QueryParam{
			Service:             cfg.TargetService,
			Domain:              "local",
			Timeout:             time.Second * 2,
			Entries:             entriesCh,
			WantUnicastResponse: false,
			Interface:           &iface,
		}
		_ = mdns.Query(params)
	}
	close(entriesCh)
}

func startProxy(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to bind port %s: %v", port, err)
	}
	log.Printf("Proxy listening on port %s", port)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error on port %s: %v", port, err)
			continue
		}

		go handleConnection(clientConn, port)
	}
}

func handleConnection(clientConn net.Conn, port string) {
	defer clientConn.Close()

	mu.RLock()
	targetHost := currentIP
	mu.RUnlock()

	if targetHost == "" {
		log.Printf("Connection rejected on port %s: Target IP unknown", port)
		return
	}

	// Dial the printer
	targetAddr := net.JoinHostPort(targetHost, port)
	backendConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		log.Printf("Failed to connect to printer at %s: %v", targetAddr, err)
		return
	}
	defer backendConn.Close()

	// Full-duplex proxy
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(backendConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(clientConn, backendConn)
		done <- struct{}{}
	}()

	<-done
}

// Helpers
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
