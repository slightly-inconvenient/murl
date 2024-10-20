package murl

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/google/cel-go/cel"
)

type InputRouteEnvironment struct {
	Allowlist []string `yaml:"allowlist" json:"allowlist"`
}

type RouteEnvironment struct {
	allowedEnvVariables map[string]bool
}

type InputRouteCheck struct {
	// Expr is a CEL expression evaluated against the request.
	// If the expression evaluates to true, the check passes.
	// If the expression evaluates to anything else, the check fails.
	Expr string `yaml:"expr" json:"expr"`

	// Error is the error message to return if the check fails.
	Error string `yaml:"error" json:"error"`
}

type RouteCheck struct {
	// Expr is a CEL expression evaluated against the request.
	// If the expression evaluates to true, the check passes.
	// If the expression evaluates to anything else, the check fails.
	expr cel.Program

	// Error is the error message to return if the check fails.
	error *template.Template
}

type InputRouteRedirect struct {
	// URL is the template to build the redirect URL from.
	URL string `yaml:"url" json:"url"`
}

type RouteRedirect struct {
	url *template.Template
}

type InputRoute struct {
	Path        string                `yaml:"path" json:"path"`
	Description string                `yaml:"description" json:"description"`
	Environment InputRouteEnvironment `yaml:"environment" json:"environment"`
	Params      map[string]string     `yaml:"params" json:"params"`
	Checks      []InputRouteCheck     `yaml:"checks" json:"checks"`
	Redirect    InputRouteRedirect    `yaml:"redirect" json:"redirect"`
}

type Route struct {
	path        string
	description string
	environment RouteEnvironment
	params      map[string]*template.Template
	checks      []RouteCheck
	redirect    RouteRedirect
	valid       bool
}

// NewRoutes parses the input routes and returns a validated route for each.
// A validated route guarantees that all required fields are present and passed all static validation such as pre-compilation of templates.
func NewRoutes(routes []InputRoute) ([]Route, error) {
	result := make([]Route, 0, len(routes))

	for idx, route := range routes {
		resultRoute := Route{
			valid:       true,
			path:        route.Path,
			description: route.Description,
		}
		if resultRoute.path == "" {
			return nil, fmt.Errorf("path to match against is required for route but missing from route at index %d", idx)
		}

		if !strings.HasPrefix(resultRoute.path, "/") {
			return nil, fmt.Errorf("path %q must be an absolute path (start with slash)", route.Path)
		}

		params, err := parseRouteParams(route.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to parse params for route %s: %w", route.Path, err)
		}
		resultRoute.params = params

		resultRoute.environment.allowedEnvVariables = parseRouteEnvAllowlist(route.Environment.Allowlist)

		celEnv, err := parseRouteCheckCelEnv(route.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to create new CEL environment for route %s: %w", route.Path, err)
		}

		for idx := range route.Checks {
			expr, err := parseRouteCheckExpr(route.Checks[idx].Expr, celEnv)
			if err != nil {
				return nil, fmt.Errorf("failed to parse check expression for route %s: %w", route.Path, err)
			}
			tmpl, err := parseTemplate(route.Checks[idx].Error)
			if err != nil {
				return nil, fmt.Errorf("failed to parse check error template for route %s: %w", route.Path, err)
			}

			resultRoute.checks = append(resultRoute.checks, RouteCheck{
				expr:  expr,
				error: tmpl,
			})
		}

		parsedURL, err := parseTemplate(route.Redirect.URL)
		resultRoute.redirect.url = parsedURL
		if err != nil {
			return nil, fmt.Errorf("failed to parse redirect url for route %s: %w", route.Path, err)
		}

		result = append(result, resultRoute)
	}

	return result, nil
}

func parseRouteParams(params map[string]string) (map[string]*template.Template, error) {
	result := make(map[string]*template.Template, len(params))
	for key, value := range params {
		parsedTemplate, err := template.New("").Parse(value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse param template %q: %w", key, err)
		}

		result[key] = parsedTemplate
	}

	return result, nil
}

func parseRouteEnvAllowlist(allowlist []string) map[string]bool {
	lookup := make(map[string]bool, len(allowlist))
	for _, key := range allowlist {
		lookup[key] = true
	}
	return lookup
}

func parseRouteCheckCelEnv(params map[string]string) (*cel.Env, error) {
	options := make([]cel.EnvOption, 0, len(params))
	for key := range params {
		options = append(options, cel.Variable(key, cel.StringType))
	}

	return cel.NewEnv(options...)
}

func parseRouteCheckExpr(expr string, env *cel.Env) (cel.Program, error) {
	if expr == "" {
		return nil, fmt.Errorf("no expression to evaluate")
	}

	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	return prg, nil
}

func parseTemplate(tmpl string) (*template.Template, error) {
	if tmpl == "" {
		return nil, fmt.Errorf("missing template")
	}

	parsedTemplate, err := template.New("").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	return parsedTemplate, nil
}
