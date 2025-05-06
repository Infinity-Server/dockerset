package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/brahma-adshonor/gohook"
	"github.com/traefik/traefik/v3/pkg/proxy"
)

const (
	ErrCloseConn = 444
)

var originalBuild func(b *proxy.SmartBuilder, configName string, targetURL *url.URL, shouldObserve, passHostHeader, preservePath bool, flushInterval time.Duration) (http.Handler, error)

func HookedBuild(b *proxy.SmartBuilder, configName string, targetURL *url.URL, shouldObserve, passHostHeader, preservePath bool, flushInterval time.Duration) (http.Handler, error) {
	log.Println("DOSK", "build")

	next, err := originalBuild(b, configName, targetURL, shouldObserve, passHostHeader, preservePath, flushInterval)
	if err != nil {
		log.Println("DOSK", "build err=", err)
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &HookedResponseWriter{
			writer: w,
		}

		log.Println("DOSK", "ServeHTTP")
		next.ServeHTTP(rw, r)
	}), nil
}

func init() {
	log.Println("DOSK", "hooked")
	gohook.Hook((*proxy.SmartBuilder).Build, HookedBuild, originalBuild)
}

type HookedResponseWriter struct {
	writer http.ResponseWriter
	http.Flusher
	http.Hijacker
}

func (rw *HookedResponseWriter) WriteHeader(statusCode int) {
	log.Println("DOSK", "WriteHeader=", statusCode)
	if statusCode == ErrCloseConn {
		log.Println("DOSK", "4444444444444444444444444444444444444444")
		hj, ok := rw.writer.(http.Hijacker)
		if !ok {
			rw.writer.WriteHeader(statusCode)
			return
		}

		conn, _, err := hj.Hijack()
		if err != nil {
			rw.writer.WriteHeader(statusCode)
			return
		}
		defer conn.Close()

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			rw.writer.WriteHeader(statusCode)
			return
		}

		tcpConn.Close()
		return
	}
	rw.writer.WriteHeader(statusCode)
}

func (rw *HookedResponseWriter) Header() http.Header {
	return rw.writer.Header()
}

func (rw *HookedResponseWriter) Write(bytes []byte) (int, error) {
	rw.WriteHeader(http.StatusOK)
	return rw.writer.Write(bytes)
}

func (rw *HookedResponseWriter) Flush() {
	if flusher, ok := rw.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *HookedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.writer.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, http.ErrHijacked
}
