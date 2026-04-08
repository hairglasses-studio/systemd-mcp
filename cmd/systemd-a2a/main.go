package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/hairglasses-studio/mcpkit/bridge/a2a"
	"github.com/hairglasses-studio/systemd-mcp/internal/systemd"
)

func main() {
	port := flag.Int("port", 8080, "Port to expose the A2A server")
	flag.Parse()

	reg, _ := systemd.Setup()

	addr := fmt.Sprintf(":%d", *port)
	url := fmt.Sprintf("http://localhost:%d", *port)

	b, err := a2a.NewBridge(reg, a2a.BridgeConfig{
		Name:        "systemd-agent",
		Description: "Systemd service and timer management agent",
		URL:         url,
		Addr:        addr,
	})
	if err != nil {
		slog.Error("failed to create a2a bridge", "error", err)
		os.Exit(1)
	}

	slog.Info("starting a2a server", "addr", addr)
	if err := b.Start(context.Background()); err != nil {
		slog.Error("a2a server stopped", "error", err)
		os.Exit(1)
	}
}
