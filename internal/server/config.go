package server

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed templates
var templates embed.FS

type TLSConfig struct {
	cert string
	key  string
}

type DocumentationConfig struct {
	path    string
	content []byte
}

type Config struct {
	address       string
	tls           TLSConfig
	documentation DocumentationConfig
	valid         bool
}

// NewConfig parses the input server configuration and returns a validated server configuration.
func NewConfig(conf config.Server, routes []config.Route) (Config, error) {
	if conf.Address == "" {
		return Config{}, fmt.Errorf("server address is required")
	}

	if conf.TLS.Cert != "" {
		if conf.TLS.Key == "" {
			return Config{}, fmt.Errorf("server TLS key is required when TLS cert is provided")
		}

		if _, err := os.Stat(conf.TLS.Cert); errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("server TLS cert file at path %q does not exist", conf.TLS.Cert)
		}
	}

	if conf.TLS.Key != "" {
		if conf.TLS.Cert == "" {
			return Config{}, fmt.Errorf("server TLS cert is required when TLS key is provided")
		}

		if _, err := os.Stat(conf.TLS.Key); errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("server TLS key file at path %q does not exist", conf.TLS.Key)
		}
	}

	documentation, err := renderDocumentation(conf.Documentation, routes)
	if err != nil {
		return Config{}, fmt.Errorf("failed to render documentation: %w", err)
	}

	return Config{
		address: conf.Address,
		tls: TLSConfig{
			cert: conf.TLS.Cert,
			key:  conf.TLS.Key,
		},
		documentation: documentation,
		valid:         true,
	}, nil
}

type docsPageHtmlInput struct {
	Content string
}

func renderDocumentation(config config.ServerDocumentationConfig, routes []config.Route) (DocumentationConfig, error) {
	tmpl := template.New("")
	for name, path := range map[string]string{
		"page":    "templates/page.html.tmpl",
		"content": "templates/content.md.tmpl",
		"routes":  "templates/routes.md.tmpl",
	} {
		content, _ := fs.ReadFile(templates, path)
		_, err := tmpl.New(name).Parse(string(content))
		if err != nil {
			return DocumentationConfig{}, fmt.Errorf("failed to parse documentation default template %q: %w", path, err)
		}
	}
	if config.Templates.Page != "" {
		if _, err := tmpl.Lookup("page").Parse(config.Templates.Page); err != nil {
			return DocumentationConfig{}, fmt.Errorf("failed to parse custom page template: %w", err)
		}
	}
	if config.Templates.Content != "" {
		if _, err := tmpl.Lookup("content").Parse(config.Templates.Content); err != nil {
			return DocumentationConfig{}, fmt.Errorf("failed to parse custom content template: %w", err)
		}
	}

	docsMarkdown := &bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(docsMarkdown, "content", routes); err != nil {
		return DocumentationConfig{}, fmt.Errorf("failed to render documentation: %w", err)
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)

	docsHtml := bytes.NewBuffer(make([]byte, 0, docsMarkdown.Len()))
	if err := markdown.Convert(docsMarkdown.Bytes(), docsHtml); err != nil {
		return DocumentationConfig{}, fmt.Errorf("failed to render documentation: %w", err)
	}

	docsPageHtml := bytes.NewBuffer(make([]byte, 0, docsHtml.Len()))
	if err := tmpl.ExecuteTemplate(docsPageHtml, "page", docsPageHtmlInput{
		Content: docsHtml.String(),
	}); err != nil {
		return DocumentationConfig{}, fmt.Errorf("failed to render documentation: %w", err)
	}

	documentationPath := "/"
	if config.Path != "" {
		documentationPath = config.Path
	}

	if !strings.HasPrefix(documentationPath, "/") {
		return DocumentationConfig{}, fmt.Errorf("documentation path must be an absolute path (start with slash)")
	}

	return DocumentationConfig{
		path:    documentationPath,
		content: docsPageHtml.Bytes(),
	}, nil
}
