package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// DocsHandler handles requests for design documents.
type DocsHandler struct {
	docsDir  string
	renderer goldmark.Markdown
}

// NewDocsHandler creates a new docs handler.
func NewDocsHandler(docsDir string) *DocsHandler {
	return &DocsHandler{
		docsDir: docsDir,
		renderer: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
		),
	}
}

// FileInfo represents a document file in the list.
type FileInfo struct {
	Name string
	Path string
}

// DocsPageData represents data for the docs template.
type DocsPageData struct {
	Title   string
	Content template.HTML
	IsIndex bool
	Files   []FileInfo
}

func (h *DocsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || path == "index" || path == "index.html" {
		h.serveIndex(w)
		return
	}

	// Clean path to prevent traversal
	fullPath := filepath.Join(h.docsDir, filepath.Clean(path))
	
	// Ensure the file is within docsDir
	absDocsDir, _ := filepath.Abs(h.docsDir)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absDocsDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) && !strings.HasSuffix(fullPath, ".md") {
			// Try adding .md extension
			fullPath += ".md"
			info, err = os.Stat(fullPath)
		}
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	if info.IsDir() {
		// If it's a directory, maybe serve an index of that directory?
		// For now, just serve the main index.
		h.serveIndex(w)
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	var buf strings.Builder
	if err := h.renderer.Convert(content, &buf); err != nil {
		http.Error(w, "Error rendering markdown", http.StatusInternalServerError)
		return
	}

	data := DocsPageData{
		Title:   filepath.Base(path),
		Content: template.HTML(buf.String()),
		IsIndex: false,
	}

	h.renderTemplate(w, data)
}

func (h *DocsHandler) serveIndex(w http.ResponseWriter) {
	var files []FileInfo
	err := filepath.Walk(h.docsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, _ := filepath.Rel(h.docsDir, path)
			// Remove .md extension and use forward slashes for cleaner URLs
			urlPath := filepath.ToSlash(strings.TrimSuffix(relPath, ".md"))
			files = append(files, FileInfo{
				Name: relPath,
				Path: "/" + urlPath,
			})
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Error listing files", http.StatusInternalServerError)
		return
	}

	data := DocsPageData{
		Title:   "Design Docs",
		IsIndex: true,
		Files:   files,
	}

	h.renderTemplate(w, data)
}

func (h *DocsHandler) renderTemplate(w http.ResponseWriter, data DocsPageData) {
	tmpl, err := template.New("docs").Parse(docsTemplate)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing template: %v\n", err)
	}
}

const docsTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Gas Town Docs</title>
    <style>
        :root {
            --bg-dark: #0f1419;
            --bg-card: #1a1f26;
            --bg-card-hover: #242b33;
            --text-primary: #e6e1cf;
            --text-secondary: #6c7680;
            --text-muted: #4a5159;
            --border: #2d363f;
            --blue: #59c2ff;
            --green: #c2d94c;
            --font-mono: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
        }

        body {
            font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
            background: var(--bg-dark);
            color: var(--text-primary);
            margin: 0;
            padding: 0;
            line-height: 1.5;
        }

        .docs-container {
            max-width: 900px;
            margin: 0 auto;
            padding: 40px 20px;
        }

        .docs-header {
            margin-bottom: 40px;
            border-bottom: 1px solid var(--border);
            padding-bottom: 20px;
        }

        .docs-header h1 {
            font-size: 1.5rem;
            margin: 0;
            color: var(--blue);
        }

        .docs-content {
            font-size: 14px;
        }

        .docs-content h1, .docs-content h2, .docs-content h3 {
            margin-top: 2em;
            margin-bottom: 1em;
            color: var(--text-primary);
            border-bottom: 1px solid var(--border);
            padding-bottom: 0.3em;
        }

        .docs-content pre {
            background: #000;
            padding: 16px;
            border-radius: 8px;
            overflow-x: auto;
            border: 1px solid var(--border);
        }

        .docs-content code {
            font-family: var(--font-mono);
            background: var(--bg-card);
            padding: 2px 4px;
            border-radius: 4px;
        }

        .docs-content a {
            color: var(--blue);
            text-decoration: none;
        }

        .docs-content a:hover {
            text-decoration: underline;
        }

        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: var(--text-secondary);
            text-decoration: none;
            font-size: 0.9rem;
        }

        .back-link:hover {
            color: var(--text-primary);
        }

        .file-list {
            list-style: none;
            padding: 0;
        }

        .file-item {
            margin-bottom: 10px;
            padding: 12px 16px;
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 6px;
            transition: background 0.2s ease;
        }

        .file-item a {
            color: var(--text-primary);
            text-decoration: none;
            display: block;
            font-weight: 500;
        }

        .file-item:hover {
            background: var(--bg-card-hover);
            border-color: var(--blue);
        }
        
        .file-item a:hover {
            color: var(--blue);
        }
    </style>
</head>
<body>
    <div class="docs-container">
        <header class="docs-header">
            {{if .IsIndex}}
            <h1>📚 Gas Town Design Docs</h1>
            {{else}}
            <a href="/" class="back-link">← Back to Index</a>
            <h1>{{.Title}}</h1>
            {{end}}
        </header>
        <main class="docs-content">
            {{if .IsIndex}}
                <ul class="file-list">
                {{range .Files}}
                    <li class="file-item"><a href="{{.Path}}">{{.Name}}</a></li>
                {{end}}
                </ul>
            {{else}}
                {{.Content}}
            {{end}}
        </main>
    </div>
</body>
</html>
`
