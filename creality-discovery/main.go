package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

type Config struct {
	TargetService string
	ProxyPorts    []string
	ScanInterval  time.Duration
	BindIface     string
}

var (
	currentIP string
	mu        sync.RWMutex
	cfg       Config
)

func info(format string, v ...interface{}) {
	ts := time.Now().Format("2006/01/02 15:04:05")
	fmt.Printf("%s [INFO] "+format+"\n", append([]interface{}{ts}, v...)...)
}

func init() {
	log.SetOutput(io.Discard) // 彻底禁言标准库
	cfg = Config{
		TargetService: fmt.Sprintf("_Creality-%s._udp", getEnv("CR_SN", "1234567890ABCD")),
		BindIface:     getEnv("CR_IFACE", "host0"),
		ProxyPorts:    strings.Split(getEnv("CR_PORTS", "4408,8000"), ","),
		ScanInterval:  time.Duration(getEnvInt("CR_INTERVAL", 10)) * time.Second,
	}
}

func main() {
	info("Starting zeroconf engine on %s", cfg.BindIface)

	// 启动发现引擎
	go discoveryLoop()

	// 启动代理
	for _, port := range cfg.ProxyPorts {
		p := strings.TrimSpace(port)
		if p != "" {
			go startProxy(p)
		}
	}

	select {}
}

func discoveryLoop() {
	for {
		// 每次扫描都重新寻找网卡，应对 K8S 网络抖动
		iface, err := net.InterfaceByName(cfg.BindIface)
		if err != nil {
			info("Wait for interface %s...", cfg.BindIface)
			time.Sleep(5 * time.Second)
			continue
		}

		// 创建 Resolver，强制绑定网卡
		resolver, err := zeroconf.NewResolver(zeroconf.SelectIfaces([]net.Interface{*iface}))
		if err != nil {
			info("Failed to create resolver: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		entries := make(chan *zeroconf.ServiceEntry)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		go func(results <-chan *zeroconf.ServiceEntry) {
			for entry := range results {
				// 提取 IPv4
				if len(entry.AddrIPv4) > 0 {
					ip := entry.AddrIPv4[0].String()
					mu.Lock()
					if currentIP != ip {
						currentIP = ip
						info("Zeroconf found printer: %s -> %s", entry.Instance, currentIP)
					}
					mu.Unlock()
				}
			}
		}(entries)

		// 开始浏览服务
		err = resolver.Browse(ctx, cfg.TargetService, "local.", entries)
		if err != nil {
			info("Browse error: %v", err)
		}

		<-ctx.Done()
		cancel()
		time.Sleep(cfg.ScanInterval)
	}
}

// --- 以下是 Proxy 部分，保持不变 ---

func startProxy(port string) {
	l, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		fmt.Printf("Fatal bind %s: %v\n", port, err)
		os.Exit(1)
	}
	info("Proxy relay ready on port %s", port)
	for {
		c, err := l.Accept()
		if err == nil {
			go handleRelay(c, port)
		}
	}
}

func handleRelay(src net.Conn, port string) {
	defer src.Close()
	mu.RLock()
	target := currentIP
	mu.RUnlock()
	if target == "" {
		return
	}
	dst, err := net.DialTimeout("tcp", net.JoinHostPort(target, port), 3*time.Second)
	if err != nil {
		return
	}
	defer dst.Close()
	done := make(chan struct{}, 2)
	go func() { io.Copy(dst, src); done <- struct{}{} }()
	go func() { io.Copy(src, dst); done <- struct{}{} }()
	<-done
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
