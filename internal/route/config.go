package route

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/cel-go/cel"
	"github.com/slightly-inconvenient/murl/internal/config"
)

type RouteEnvironment struct {
	allowedEnvVariables map[string]bool
}

type RouteCheck struct {
	// Expr is a CEL expression evaluated against the request.
	// If the expression evaluates to true, the check passes.
	// If the expression evaluates to anything else, the check fails.
	expr cel.Program

	// Error is the error message to return if the check fails.
	error *template.Template
}

type RouteRedirect struct {
	url *template.Template
}

type RouteTestRequest struct {
	environment map[string]string
	headers     map[string]string
	url         string
}

type RouteTestResponse struct {
	status int
	url    string
}

type RouteTest struct {
	request  RouteTestRequest
	response RouteTestResponse
}

type Route struct {
	paths       []string
	environment RouteEnvironment
	params      map[string]*template.Template
	checks      []RouteCheck
	redirect    RouteRedirect
	tests       []RouteTest
	valid       bool
}

// NewRoutes parses the input routes and returns a validated route for each.
// A validated route guarantees that all required fields are present and passed all static validation such as pre-compilation of templates.
func NewRoutes(routes []config.Route) ([]Route, error) {
	result := make([]Route, 0, len(routes))

	for idx, route := range routes {
		resultRoute := Route{
			valid: true,
		}

		paths, err := parseRoutePaths(route.Path, route.Aliases)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path or alias for route at index [%d]: %w", idx, err)
		}
		resultRoute.paths = paths

		params, err := parseRouteParams(route.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to parse params for route at index [%d]: %w", idx, err)
		}
		resultRoute.params = params

		resultRoute.environment.allowedEnvVariables = parseRouteEnvAllowlist(route.Environment.Allowlist)

		celEnv, err := parseRouteCheckCelEnv(route.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to create new CEL environment for route at index [%d]: %w", idx, err)
		}

		for idx := range route.Checks {
			expr, err := parseRouteCheckExpr(route.Checks[idx].Expr, celEnv)
			if err != nil {
				return nil, fmt.Errorf("failed to parse check expression for route at index [%d]: %w", idx, err)
			}
			tmpl, err := parseTemplate(route.Checks[idx].Error)
			if err != nil {
				return nil, fmt.Errorf("failed to parse check error template for route at index [%d]: %w", idx, err)
			}

			resultRoute.checks = append(resultRoute.checks, RouteCheck{
				expr:  expr,
				error: tmpl,
			})
		}

		for tidx, test := range route.Tests {
			resultTest, err := parseTest(test)
			if err != nil {
				return nil, fmt.Errorf("failed to parse test [%d] for route at index [%d]: %w", tidx, idx, err)
			}
			resultRoute.tests = append(resultRoute.tests, resultTest)
		}

		parsedURL, err := parseTemplate(route.Redirect.URL)
		resultRoute.redirect.url = parsedURL
		if err != nil {
			return nil, fmt.Errorf("failed to parse redirect url for route at index [%d]: %w", idx, err)
		}

		result = append(result, resultRoute)
	}

	return result, nil
}

func parseRoutePaths(path string, aliases []string) ([]string, error) {
	paths := make([]string, 0, len(aliases)+1)
	paths = append(paths, path)
	paths = append(paths, aliases...)
	for _, path := range paths {
		if !strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("%q must be an absolute path (start with slash)", path)
		}
	}

	return paths, nil
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

func parseTest(test config.RouteTest) (RouteTest, error) {
	result := RouteTest{
		request: RouteTestRequest{
			url:         test.Request.URL,
			headers:     test.Request.Headers,
			environment: test.Request.Environment,
		},
		response: RouteTestResponse{
			status: http.StatusTemporaryRedirect,
			url:    test.Response.URL,
		},
	}

	if result.request.url == "" {
		return RouteTest{}, fmt.Errorf("test request url is required but was missing")
	}
	if result.response.url == "" {
		return RouteTest{}, fmt.Errorf("test response url is required but was missing")
	}

	return result, nil
}
