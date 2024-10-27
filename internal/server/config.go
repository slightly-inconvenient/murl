package server

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/slightly-inconvenient/murl"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed templates
var templates embed.FS

type InputTLSConfig struct {
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

type TLSConfig struct {
	cert string
	key  string
}

type InputTemplatesConfig struct {
	Root string `yaml:"root" json:"root"`
}

type InputDocumentationConfig struct {
	Path      string               `yaml:"path" json:"path"`
	Templates InputTemplatesConfig `yaml:"templates" json:"templates"`
}

type InputConfig struct {
	Address       string                   `yaml:"address" json:"address"`
	TLS           InputTLSConfig           `yaml:"tls" json:"tls"`
	Documentation InputDocumentationConfig `yaml:"documentation" json:"documentation"`
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
func NewConfig(config InputConfig, routes []murl.InputRoute) (Config, error) {
	if config.Address == "" {
		return Config{}, fmt.Errorf("server address is required")
	}

	if config.TLS.Cert != "" {
		if config.TLS.Key == "" {
			return Config{}, fmt.Errorf("server TLS key is required when TLS cert is provided")
		}

		if _, err := os.Stat(config.TLS.Cert); errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("server TLS cert file at path %q does not exist", config.TLS.Cert)
		}
	}

	if config.TLS.Key != "" {
		if config.TLS.Cert == "" {
			return Config{}, fmt.Errorf("server TLS cert is required when TLS key is provided")
		}

		if _, err := os.Stat(config.TLS.Key); errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("server TLS key file at path %q does not exist", config.TLS.Key)
		}
	}

	documentation, err := renderDocumentation(config.Documentation, routes)
	if err != nil {
		return Config{}, fmt.Errorf("failed to render documentation: %w", err)
	}

	return Config{
		address: config.Address,
		tls: TLSConfig{
			cert: config.TLS.Cert,
			key:  config.TLS.Key,
		},
		documentation: documentation,
		valid:         true,
	}, nil
}

func renderDocumentation(config InputDocumentationConfig, routes []murl.InputRoute) (DocumentationConfig, error) {
	tmpl, err := template.ParseFS(templates, "templates/*")
	if err != nil {
		return DocumentationConfig{}, fmt.Errorf("failed to parse documentation default templates: %w", err)
	}
	if config.Templates.Root != "" {
		if _, err := tmpl.Lookup("root").Parse(config.Templates.Root); err != nil {
			return DocumentationConfig{}, fmt.Errorf("failed to parse custom root template: %w", err)
		}
	}

	docsMarkdown := &bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(docsMarkdown, "root", routes); err != nil {
		return DocumentationConfig{}, fmt.Errorf("failed to render documentation: %w", err)
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	docsHtml := bytes.NewBuffer(make([]byte, 0, docsMarkdown.Len()))
	if err := markdown.Convert(docsMarkdown.Bytes(), docsHtml); err != nil {
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
		content: docsHtml.Bytes(),
	}, nil
}
