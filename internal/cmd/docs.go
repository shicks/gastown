package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/web"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	docsPort int
	docsBind string
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Start a web server to serve design documents",
	Long: `Start a web server that renders and serves design documents from the docs/ directory.

Example:
  gt docs              # Start on default port 2001
  gt docs --port 3000  # Start on port 3000`,
	RunE: runDocs,
}

func init() {
	docsCmd.Flags().IntVar(&docsPort, "port", 2001, "HTTP port to listen on")
	docsCmd.Flags().StringVar(&docsBind, "bind", "127.0.0.1", "Address to bind to")
	rootCmd.AddCommand(docsCmd)
}

func runDocs(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Try to find docs/ directory by walking up from CWD
	var docsDir string
	curr := cwd
	for {
		dir := filepath.Join(curr, "docs")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			docsDir = dir
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	if docsDir == "" {
		// Fallback to town root if not found by walking up
		townRoot, err := workspace.FindFromCwdOrError()
		if err != nil {
			return fmt.Errorf("finding workspace: %w", err)
		}
		docsDir = filepath.Join(townRoot, "docs")
	}
	handler := web.NewDocsHandler(docsDir)

	listenAddr := fmt.Sprintf("%s:%d", docsBind, docsPort)
	fmt.Printf("📚 serving design docs at http://%s  •  ctrl+c to stop\n", listenAddr)

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return server.ListenAndServe()
}
