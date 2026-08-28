// Package mcpserver assembles the MCP server behind `fizzy mcp`: the
// hand-written Fizzy tool catalog served through the shared toolkit's
// gateway, dispatching real API calls through the CLI's authenticated,
// account-scoped SDK client.
//
// The generic machinery — domain gateway tools, the {"action", "params"}
// calling convention, read-only filtering, the in-band describe action —
// lives in the shared toolkit at github.com/basecamp/mcp. The catalog in
// internal/mcpserver/catalog is deliberately duplicated from its sibling
// in fizzy-mcp-server (synced by scripts/sync-mcp-catalog.sh, provenance
// recorded). This package supplies the CLI's half: wiring the catalog to
// the gateway and the dispatcher that turns catalog operations into
// fizzy-sdk requests.
package mcpserver

import (
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/mcp/gateway"

	"github.com/basecamp/fizzy-cli/internal/mcpserver/catalog"
)

// Name identifies the server in the MCP initialize handshake. The version
// is the CLI's own: `fizzy mcp` is the CLI serving MCP, not a separate
// product.
const Name = "fizzy-cli"

// Config selects the served tool surface.
type Config struct {
	// ReadOnly drops every write action from the catalog and refuses write
	// dispatch outright. The served default, matching fizzy-mcp-server's
	// posture: writes are an explicit opt-in, paired with a Read+Write
	// token — the token's permission is the server-side enforcement, this
	// filter is the client-side surface.
	ReadOnly bool
	// Domains narrows the served domains by key ("boards", "cards", ...).
	// Empty means all. Unknown keys are a startup error — fail closed.
	Domains []string
	// Version is the CLI version reported in the initialize handshake.
	Version string
}

// Server wraps the toolkit gateway serving the catalog, dispatching
// through the CLI's SDK clients.
type Server struct {
	gw      *gateway.Server
	version string
}

// New validates the catalog and hands it to the gateway, which applies
// the config's domain and read-only filters. Tool calls dispatch through
// account (account-scoped operations) and root (the few unscoped ones,
// like get_identity).
func New(account, root API, cfg Config) (*Server, error) {
	if account == nil || root == nil {
		return nil, fmt.Errorf("mcpserver: account and root API clients are required")
	}

	domains, err := catalog.Load()
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}

	srv := &Server{version: cfg.Version}
	gw, err := gateway.New(catalog.GatewayDomains(domains), gateway.Config{
		ReadOnly: cfg.ReadOnly,
		Domains:  cfg.Domains,
		Handler:  dispatcher{account: account, root: root}.handle,
	})
	if err != nil {
		return nil, err
	}
	srv.gw = gw

	return srv, nil
}

// Domains returns the served domains.
func (s *Server) Domains() []gateway.Domain {
	return s.gw.Domains()
}

// BuildMCPServer constructs the SDK MCP server with one gateway tool per
// served domain.
func (s *Server) BuildMCPServer(logger *slog.Logger) *mcp.Server {
	return s.gw.BuildMCPServer(&mcp.Implementation{Name: Name, Version: s.version}, logger)
}
