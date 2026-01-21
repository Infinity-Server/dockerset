package main

import (
	"os"

	"geo-sni-proxy/internal/app"
)

func main() {
	if err := app.Start(); err != nil {
		os.Exit(1)
	}
	select {}
}
