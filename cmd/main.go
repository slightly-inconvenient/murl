package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/route"
	"github.com/slightly-inconvenient/murl/internal/server"
	"github.com/urfave/cli/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	os.Exit(run(ctx))
}

func run(ctx context.Context) int {
	cmd := &cli.Command{
		Name:                  "murl",
		Usage:                 "Templated redirects",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			createServeCommand(),
			createValidateCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Usage:    "Path to the configuration file",
				Required: true,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("MURL_CONFIG"),
				),
				Value: "config.json",
			},
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}

func createServeCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start a server to serve the configured redirect routes",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			conf, err := config.ParseConfigFile(cmd.String("config"))
			if err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			serverConfig, err := server.NewConfig(conf.Server, conf.Routes)
			if err != nil {
				return fmt.Errorf("invalid server config: %w", err)
			}

			routes, err := route.NewRoutes(conf.Routes)
			if err != nil {
				return fmt.Errorf("invalid routes: %w", err)
			}

			if err := server.Run(ctx, serverConfig, route.NewHandlers(routes)); err != nil {
				return fmt.Errorf("failed to serve: %w", err)
			}

			return nil
		},
	}
}

func createValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate routes against tests defined in them",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			conf, err := config.ParseConfigFile(cmd.String("config"))
			if err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			routes, err := route.NewRoutes(conf.Routes)
			if err != nil {
				return fmt.Errorf("invalid routes: %w", err)
			}

			handlers := route.NewHandlers(routes)
			if err := route.TestHandlers(ctx, routes, handlers); err != nil {
				return fmt.Errorf("failed tests: %w", err)
			}

			return nil
		},
	}
}
