package main

import (
  "context"
  "os/signal"
  "syscall"
)

func main() {
  ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
  defer cancel()

  xunlei := XunleiDaemon{}
  xunlei.Run(ctx)
}
