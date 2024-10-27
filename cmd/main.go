package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/route"
	"github.com/slightly-inconvenient/murl/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	os.Exit(run(ctx))
}

func run(ctx context.Context) int {
	flags := createFlags()
	if err := flags.Parse(os.Args[1:]); err != nil {
		return handleError(err)
	}

	conf, err := config.ParseConfigFile(flags.Lookup("config").Value.String())
	if err != nil {
		return handleError(err)
	}

	serverConfig, err := server.NewConfig(conf.Server, conf.Routes)
	if err != nil {
		return handleError(err)
	}

	routes, err := route.NewRoutes(conf.Routes)
	if err != nil {
		return handleError(err)
	}

	if err := server.Run(ctx, serverConfig, route.NewHandlers(routes)); err != nil {
		return handleError(err)
	}

	return 0
}

func createFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("murl", flag.ExitOnError)
	flags.Usage = func() {
		flags.PrintDefaults()
	}
	flags.String("config", "config.json", "Path to the configuration file")
	return flags
}

func handleError(err error) int {
	fmt.Println(err)
	return 1
}
