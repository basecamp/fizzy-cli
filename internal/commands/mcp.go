package commands

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/basecamp/fizzy-cli/internal/mcpserver"
)

// mcpTransport is a seam so tests can drive the server over in-memory
// transports instead of the process's stdin/stdout.
var mcpTransport = func() mcp.Transport { return &mcp.StdioTransport{} }

var (
	mcpWrites  bool
	mcpDomains []string
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve Fizzy to MCP clients over stdio",
	Long: "Run an MCP (Model Context Protocol) server on stdin/stdout, serving Fizzy\n" +
		"boards, columns, cards, comments, steps, tags, users, and your identity as\n" +
		"tools backed by your signed-in account.\n\n" +
		"Read-only by default; --writes serves write actions too (pair with a\n" +
		"Read+Write access token). Register it with an MCP client as a stdio\n" +
		"server, e.g.:\n\n" +
		"  claude mcp add fizzy -- fizzy mcp",
	Args: cobra.NoArgs,
	Annotations: map[string]string{
		"agent_notes": "Long-running server; stdout speaks the MCP wire protocol. Not for interactive use.",
	},
	RunE: runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	if err := requireAuthAndAccount(); err != nil {
		return err
	}

	srv, err := mcpserver.New(getSDK(), getSDKClient(), mcpserver.Config{
		ReadOnly: !mcpWrites,
		Domains:  mcpDomains,
		Version:  currentVersion(),
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Log to stderr: stdout belongs to the MCP wire.
	logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))
	session, err := srv.BuildMCPServer(logger).Connect(ctx, mcpTransport(), nil)
	if err != nil {
		return err
	}
	logger.Info("MCP server running on stdio", "tools", len(srv.Domains()), "read_only", !mcpWrites)

	return session.Wait()
}

func init() {
	mcpCmd.Flags().BoolVar(&mcpWrites, "writes", false, "Serve write actions too (pair with a Read+Write access token)")
	mcpCmd.Flags().StringSliceVar(&mcpDomains, "domains", nil, "Narrow to specific domains (comma-separated; default all)")
	rootCmd.AddCommand(mcpCmd)
}
