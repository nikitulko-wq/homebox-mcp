// homebox-mcp is an MCP server exposing Homebox (v0.26+) inventory management
// to MCP clients over stdio.
//
// Configuration via environment variables:
//
//	HOMEBOX_URL       base URL of the Homebox instance, e.g. https://homebox.example.com
//	HOMEBOX_EMAIL     login email (Homebox username)
//	HOMEBOX_PASSWORD  login password
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/litvinovns/homebox-mcp/internal/config"
	"github.com/litvinovns/homebox-mcp/internal/homebox"
	"github.com/litvinovns/homebox-mcp/internal/tools"
)

// version is overridden at release build time via -ldflags "-X main.version=...".
var version = "1.0.0"

func main() {
	// stdout is reserved for the MCP stdio protocol — log only to stderr.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	cfg, err := config.Load()
	if err != nil {
		log.Printf("configuration error: %v", err)
		log.Print("set HOMEBOX_URL, HOMEBOX_EMAIL and HOMEBOX_PASSWORD environment variables")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hb := homebox.NewClient(cfg.URL, cfg.Email, cfg.Password)
	if err := hb.Login(ctx); err != nil {
		log.Printf("warning: initial login failed: %v (will retry lazily on first tool call)", err)
	} else if st, err := hb.Status(ctx); err == nil {
		log.Printf("connected to Homebox %s at %s", st.Build.Version, cfg.URL)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "homebox-mcp",
		Version: version,
	}, nil)
	tools.Register(server, hb)

	log.Printf("starting homebox-mcp %s (stdio)", version)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
