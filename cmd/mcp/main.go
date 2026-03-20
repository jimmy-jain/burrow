// Package main provides the burrow-mcp binary, an MCP server that exposes
// burrow CLI commands as tools for Claude integration via stdio transport.
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "burrow"
	serverVersion = "0.2.1"
)

func main() {
	exec, err := NewExecutor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "burrow-mcp: %v\n", err)
		os.Exit(1)
	}

	srv := server.NewMCPServer(serverName, serverVersion,
		server.WithToolCapabilities(false),
	)

	registerTools(srv, exec)

	if err := server.ServeStdio(srv); err != nil {
		fmt.Fprintf(os.Stderr, "burrow-mcp: stdio server error: %v\n", err)
		os.Exit(1)
	}
}
