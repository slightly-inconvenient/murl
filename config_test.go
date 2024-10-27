package murl_test

import (
	"errors"
	"testing"

	"github.com/slightly-inconvenient/murl"
)

type routeTestTransform func(*murl.InputRoute)

func buildTestRoute(transforms ...routeTestTransform) murl.InputRoute {
	result := murl.InputRoute{
		Path:    "/example/{rest}",
		Aliases: []string{"/example2/{rest}"},
		Documentation: murl.InputRouteDocumentation{
			Title:       "Example Route",
			Description: "An example route that picks params from the env, path, query, and headers",
		},
		Environment: murl.InputRouteEnvironment{
			Allowlist: []string{"EXAMPLE_HOST"},
		},
		Params: map[string]string{
			"path":        `{{.GetParam "rest"}}`,
			"host":        `{{.GetEnv "EXAMPLE_HOST"}}`,
			"query":       `{{.GetQuery "query"}}`,
			"contentType": `{{.GetHeader "content-type"}}`,
		},
		Checks: []murl.InputRouteCheck{
			{
				Expr:  `host != ""`,
				Error: "host is required",
			},
			{
				Expr:  `path != ""`,
				Error: "path is required",
			},
		},
		Redirect: murl.InputRouteRedirect{
			URL: "https://{{.host}}/any/will/do/{{.path}}?q={{.query}}&h={{.contentType}}",
		},
	}

	for _, transform := range transforms {
		transform(&result)
	}

	return result
}

func TestConfig_Success(t *testing.T) {
	t.Parallel()
	input := buildTestRoute()
	routes, err := murl.NewRoutes([]murl.InputRoute{input})
	if err != nil {
		t.Fatalf("expected create routes to succeed but got error: %s", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route but got %d", len(routes))
	}
}

func TestConfig_Failures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		description   string
		route         murl.InputRoute
		expectedError error
	}{
		{
			description: "fails with non-absolute path",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Path = "example"
			}),
			expectedError: errors.New("failed to parse path or alias for route at index [0]: \"example\" must be an absolute path (start with slash)"),
		},
		{
			description: "fails with non-absolute path alias",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Aliases = []string{"example2"}
			}),
			expectedError: errors.New("failed to parse path or alias for route at index [0]: \"example2\" must be an absolute path (start with slash)"),
		},
		{
			description: "fails with bad params input",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Params["id"] = "{{{}}"
			}),
			expectedError: errors.New("failed to parse params for route at index [0]: failed to parse param template \"id\": template: :1: unexpected \"{\" in command"),
		},
		{
			description: "fails with bad cel expr",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Checks[0].Expr = "{"
			}),
			expectedError: errors.New("failed to parse check expression for route at index [0]: ERROR: <input>:1:2: Syntax error: mismatched input '<EOF>' expecting {'[', '{', '}', '(', '.', ',', '-', '!', '?', 'true', 'false', 'null', NUM_FLOAT, NUM_INT, NUM_UINT, STRING, BYTES, IDENTIFIER}\n | {\n | .^"),
		},
		{
			description: "fails with missing cel expr",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Checks[0].Expr = ""
			}),
			expectedError: errors.New("failed to parse check expression for route at index [0]: no expression to evaluate"),
		},
		{
			description: "fails with missing cel error",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Checks[0].Error = ""
			}),
			expectedError: errors.New("failed to parse check error template for route at index [0]: missing template"),
		},
		{
			description: "fails with invalid cel error template",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Checks[0].Error = "{{{}}"
			}),
			expectedError: errors.New("failed to parse check error template for route at index [0]: template: :1: unexpected \"{\" in command"),
		},
		{
			description: "fails with bad redirect url template",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Redirect.URL = "{{{}}"
			}),
			expectedError: errors.New("failed to parse redirect url for route at index [0]: template: :1: unexpected \"{\" in command"),
		},
		{
			description: "fails with missing redirect url template",
			route: buildTestRoute(func(route *murl.InputRoute) {
				route.Redirect.URL = ""
			}),
			expectedError: errors.New("failed to parse redirect url for route at index [0]: missing template"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			_, err := murl.NewRoutes([]murl.InputRoute{test.route})
			if err == nil {
				t.Fatalf("expected create routes to fail but got nil")
			}
			if err.Error() != test.expectedError.Error() {
				t.Fatalf("expected error %q but got %q", test.expectedError, err)
			}
		})
	}
}
